#!/bin/bash

# Default binary name
DEFAULT_NAME="deploygo"

# Prompt for binary name
read -p "Enter binary name [${DEFAULT_NAME}]: " BINARY_NAME
BINARY_NAME=${BINARY_NAME:-$DEFAULT_NAME}

echo ""
echo "Select build targets:"
echo "1) ARM64 (Linux arm64 + macOS arm64)"
echo "2) x64   (Linux amd64 + macOS amd64)"
echo "3) Both  (All of the above)"
echo "4) Current System only"
echo "5) Custom..."

read -p "Enter choice [4]: " CHOICE
CHOICE=${CHOICE:-4}

mkdir -p build

build_binary() {
    local os=$1
    local arch=$2
    local output_name="build/${BINARY_NAME}-${os}-${arch}"
    
    # Windows extension
    if [ "$os" == "windows" ]; then
        output_name="${output_name}.exe"
    fi

    echo "Building for ${os}/${arch}..."
    GOOS=$os GOARCH=$arch go build -o "$output_name"
    if [ $? -eq 0 ]; then
        echo "✓ Created $output_name"
    else
        echo "✗ Failed to build for ${os}/${arch}"
    fi
}

case $CHOICE in
    1)
        build_binary linux arm64
        build_binary darwin arm64
        ;;
    2)
        build_binary linux amd64
        build_binary darwin amd64
        ;;
    3)
        build_binary linux arm64
        build_binary darwin arm64
        build_binary linux amd64
        build_binary darwin amd64
        ;;
    4)
        echo "Building for current system..."
        go build -o "build/${BINARY_NAME}"
        echo "✓ Created build/${BINARY_NAME}"
        ;;
    5)
        echo ""
        echo "Select specific platform:"
        echo "1) Linux amd64"
        echo "2) Linux arm64"
        echo "3) macOS amd64 (Intel)"
        echo "4) macOS arm64 (Apple Silicon)"
        echo "5) Windows amd64"
        read -p "Enter choice: " CUSTOM_CHOICE
        case $CUSTOM_CHOICE in
            1) build_binary linux amd64 ;;
            2) build_binary linux arm64 ;;
            3) build_binary darwin amd64 ;;
            4) build_binary darwin arm64 ;;
            5) build_binary windows amd64 ;;
            *) echo "Invalid choice"; exit 1 ;;
        esac
        ;;
    *)
        echo "Invalid choice"
        exit 1
        ;;
esac

echo ""
echo "Build complete! Check the 'build' directory."
