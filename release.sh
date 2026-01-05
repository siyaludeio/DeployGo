#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# release.sh
#
# Creates an annotated git tag, pushes it, then creates a GitHub Release as DRAFT
# using the annotated tag message as the release body, uploads dist/*.tar.gz
# assets, and optionally publishes the release.
#
# Requirements:
#   - git
#   - curl
#   - jq
#   - GitHub auth already available (NO prompt):
#       * either export GITHUB_TOKEN=...
#       * or be logged in via `gh auth login` (token is reused automatically)
# ==============================================================================

PROJECT_NAME="${PROJECT_NAME:-DeployGo}"
DIST_DIR="${DIST_DIR:-dist}"

# ----------------------------- helpers ----------------------------------------

die() { echo "❌ $*" >&2; exit 1; }

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "Missing required command: $1"
}

prompt_default() {
  # usage: prompt_default "Question" "default"
  local q="$1" d="$2" ans
  read -r -p "$q [$d]: " ans || true
  if [[ -z "${ans:-}" ]]; then
    echo "$d"
  else
    echo "$ans"
  fi
}

prompt_yes_no() {
  # usage: prompt_yes_no "Question" "N|Y"
  local q="$1" default="$2" ans
  read -r -p "$q (${default}/$( [[ "$default" == "Y" ]] && echo "N" || echo "Y" )): " ans || true
  ans="${ans:-}"
  if [[ -z "$ans" ]]; then
    [[ "$default" == "Y" ]] && return 0 || return 1
  fi
  [[ "$ans" =~ ^[Yy]$ ]]
}

resolve_github_token() {
  # 1) explicit env
  if [[ -n "${GITHUB_TOKEN:-}" ]]; then
    echo "$GITHUB_TOKEN"
    return 0
  fi

  # 2) gh cli token (silent, no prompts)
  if command -v gh >/dev/null 2>&1; then
    if gh auth status >/dev/null 2>&1; then
      gh auth token
      return 0
    fi
  fi

  # 3) parse gh hosts.yml (same idea as many small helper scripts)
  local hosts="$HOME/.config/gh/hosts.yml"
  if [[ -f "$hosts" ]]; then
    # first oauth_token found
    local tok
    tok="$(awk '/oauth_token:/ { print $2; exit }' "$hosts" || true)"
    if [[ -n "${tok:-}" ]]; then
      echo "$tok"
      return 0
    fi
  fi

  return 1
}

detect_github_repo_slug() {
  # returns "owner/repo" from origin remote URL (ssh or https)
  local url
  #url="$(git remote get-url origin 2>/dev/null || true)"
  url="https://github.com/siyaludeio/DeployGo"
  [[ -n "${url:-}" ]] || die "Cannot determine git origin remote. Is this a git repo with an 'origin' remote?"

  # Supports:
  #   git@github.com:owner/repo.git
  #   https://github.com/owner/repo.git
  #   https://github.com/owner/repo
  #local slug
  #slug="$(echo "$url" | sed -E 's#(git@|https://)github.com[:/](.+/.+?)(\.git)?$#\2#' || true)"

  #[[ "$slug" == */* ]] || die "Origin remote is not a GitHub URL: $url"
  echo "siyaludeio/DeployGo"
}

semver_bump_patch() {
  # Input: vX.Y.Z or X.Y.Z
  # Output: vX.Y.(Z+1)
  local v="$1"
  v="${v#v}"
  local IFS=.
  read -r major minor patch <<<"$v" || true
  [[ "${major:-}" =~ ^[0-9]+$ && "${minor:-}" =~ ^[0-9]+$ && "${patch:-}" =~ ^[0-9]+$ ]] || return 1
  patch=$((patch + 1))
  echo "v${major}.${minor}.${patch}"
}

# ----------------------------- preflight --------------------------------------

need_cmd git
need_cmd curl
need_cmd jq

git rev-parse --is-inside-work-tree >/dev/null 2>&1 || die "Not inside a git repository."

if ! git diff --quiet || ! git diff --cached --quiet; then
  echo "⚠️  Working tree has uncommitted changes."
  if ! prompt_yes_no "Continue anyway?" "N"; then
    die "Aborted."
  fi
fi

# ----------------------------- versioning -------------------------------------

LATEST_TAG="$(git describe --tags --abbrev=0 2>/dev/null || true)"

if [[ -z "${LATEST_TAG:-}" ]]; then
  echo "No tags found in repository."
  SUGGESTED_VERSION="v0.1.0"
else
  echo "Current latest version: $LATEST_TAG"
  SUGGESTED_VERSION="$(semver_bump_patch "$LATEST_TAG" || echo "v0.1.0")"
fi

RELEASE_VERSION="$(prompt_default "Enter new release version (must start with 'v')" "$SUGGESTED_VERSION")"
[[ "$RELEASE_VERSION" =~ ^v[0-9]+(\.[0-9]+){2}([\-+][0-9A-Za-z\.\-]+)?$ ]] || die "Invalid version format: $RELEASE_VERSION (expected vX.Y.Z)"

if git rev-parse -q --verify "refs/tags/$RELEASE_VERSION" >/dev/null; then
  die "Tag already exists locally: $RELEASE_VERSION"
fi

echo ""
echo "📝 Release notes (will become the annotated tag message, and GitHub release body)"
echo "   Finish input with CTRL+D (or paste notes then CTRL+D)."
echo "--------------------------------------------------------"
RELEASE_NOTE="$(cat || true)"
[[ -n "${RELEASE_NOTE// }" ]] || RELEASE_NOTE="Release $RELEASE_VERSION"

# ----------------------------- build artifacts check ---------------------------

if [[ ! -d "$DIST_DIR" ]]; then
  echo "⚠️  Dist directory '$DIST_DIR' not found."
else
  shopt -s nullglob
  ASSETS=( "$DIST_DIR"/*.tar.gz )
  shopt -u nullglob
  if [[ ${#ASSETS[@]} -eq 0 ]]; then
    echo "⚠️  No assets found at '$DIST_DIR/*.tar.gz'."
    echo "   (The script will still create the tag and draft release, but nothing will be uploaded.)"
  fi
fi

# ----------------------------- tag + push -------------------------------------

echo ""
echo "🏷️  Creating annotated git tag: $RELEASE_VERSION"
git tag -a "$RELEASE_VERSION" -m "$RELEASE_NOTE"

echo "⬆️  Pushing tag to remote..."
git push origin "$RELEASE_VERSION"

# ----------------------------- GitHub release ---------------------------------

REPO_SLUG="$(detect_github_repo_slug)"
GITHUB_API="https://api.github.com/repos/$REPO_SLUG"

GITHUB_TOKEN_RESOLVED="$(resolve_github_token || true)"
[[ -n "${GITHUB_TOKEN_RESOLVED:-}" ]] || die "No GitHub token found. Either export GITHUB_TOKEN or run 'gh auth login' once."

AUTH_HEADER="Authorization: Bearer $GITHUB_TOKEN_RESOLVED"

TAG="$RELEASE_VERSION"
RELEASE_BODY="$(git for-each-ref "refs/tags/$TAG" --format='%(contents)' || true)"
[[ -n "${RELEASE_BODY// }" ]] || RELEASE_BODY="Release $TAG"

echo ""
echo "🧹 Idempotency: deleting existing GitHub release for tag '$TAG' (if any)..."

EXISTING_RELEASE_JSON="$(curl -sS -H "$AUTH_HEADER" -H "Accept: application/vnd.github+json" \
  "$GITHUB_API/releases/tags/$TAG" || true)"

EXISTING_RELEASE_ID="$(echo "$EXISTING_RELEASE_JSON" | jq -r '.id // empty' || true)"

if [[ -n "${EXISTING_RELEASE_ID:-}" ]]; then
  curl -sS -X DELETE -H "$AUTH_HEADER" -H "Accept: application/vnd.github+json" \
    "$GITHUB_API/releases/$EXISTING_RELEASE_ID" >/dev/null
  echo "✅ Deleted existing release (ID: $EXISTING_RELEASE_ID)"
else
  echo "✅ No existing release found for $TAG"
fi

echo ""
echo "📦 Creating GitHub release as DRAFT..."

CREATE_RELEASE_PAYLOAD="$(jq -n \
  --arg tag "$TAG" \
  --arg name "$TAG" \
  --arg body "$RELEASE_BODY" \
  '{
    tag_name: $tag,
    name: $name,
    body: $body,
    draft: true,
    prerelease: false
  }'
)"

CREATE_RELEASE_RESPONSE="$(curl -sS -X POST "$GITHUB_API/releases" \
  -H "$AUTH_HEADER" \
  -H "Accept: application/vnd.github+json" \
  -d "$CREATE_RELEASE_PAYLOAD")"

RELEASE_ID="$(echo "$CREATE_RELEASE_RESPONSE" | jq -r '.id // empty')"
UPLOAD_URL="$(echo "$CREATE_RELEASE_RESPONSE" | jq -r '.upload_url // empty' | sed 's/{?name,label}//')"
HTML_URL="$(echo "$CREATE_RELEASE_RESPONSE" | jq -r '.html_url // empty')"

[[ -n "${RELEASE_ID:-}" ]] || die "Failed to create GitHub release. Response: $CREATE_RELEASE_RESPONSE"
echo "✅ Draft release created: ${HTML_URL:-"(no url)"}"

# ----------------------------- upload assets ----------------------------------

if [[ -d "$DIST_DIR" ]]; then
  shopt -s nullglob
  ASSETS=( "$DIST_DIR"/*.tar.gz )
  shopt -u nullglob

  if [[ ${#ASSETS[@]} -gt 0 ]]; then
    echo ""
    echo "⬆️  Uploading assets to release..."
    for file in "${ASSETS[@]}"; do
      fname="$(basename "$file")"
      echo "   - $fname"
      curl -sS -X POST "${UPLOAD_URL}?name=$(printf '%s' "$fname" | jq -sRr @uri)" \
        -H "$AUTH_HEADER" \
        -H "Accept: application/vnd.github+json" \
        -H "Content-Type: application/gzip" \
        --data-binary @"$file" >/dev/null
    done
    echo "✅ Assets uploaded"
  else
    echo "ℹ️  No assets to upload."
  fi
fi

# ----------------------------- publish prompt ---------------------------------

echo ""
echo "------------------------------------------------"
echo "🎉 Draft release ready for $TAG"
[[ -n "${HTML_URL:-}" ]] && echo "🔗 ${HTML_URL}"
echo "------------------------------------------------"

if prompt_yes_no "🚀 Publish this release now?" "N"; then
  curl -sS -X PATCH "$GITHUB_API/releases/$RELEASE_ID" \
    -H "$AUTH_HEADER" \
    -H "Accept: application/vnd.github+json" \
    -d '{"draft":false}' >/dev/null
  echo "🎉 Release published!"
else
  echo "📝 Left in DRAFT mode."
fi
