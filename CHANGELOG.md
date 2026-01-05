# Changelog

All notable changes to this project will be documented in this file.

## [v1.0.0] - 2026-01-06

### 🚀 Initial Release

**DeployGo** (formerly self-deployer) is officially released as a standalone CLI tool.

#### Features
- **Detached Execution**: Complete background processing using self-spawning subprocesses.
- **Cross-Platform**: Builds for Linux (amd64/arm64) and macOS (amd64/arm64).
- **Secure Logging**: Implemented non-blocking logging with automatic rotation.
- **Strict Validation**: Enforced absolute paths for security.

#### Breaking Changes from Prototype
- Renamed binary from `self-deployer` to `deploygo`.
- Removed HTTP server/listener components in favor of CLI-only usage.
- Removed legacy `deployer.service` systemd files.
