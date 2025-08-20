#!/bin/bash

# Build script for k6 with penelopa output module
# Usage: ./build.sh [linux|darwin] [amd64|arm64]

set -e

# Default values
OS=${1:-linux}
ARCH=${2:-amd64}
K6_VERSION=${K6_VERSION:-v0.46.0}
OUTPUT_NAME="k6-penelopa-${OS}-${ARCH}"

echo "Building k6 with penelopa output module..."
echo "OS: $OS"
echo "ARCH: $ARCH"
echo "K6 Version: $K6_VERSION"
echo "Output: $OUTPUT_NAME"

# Build command
GOOS=$OS GOARCH=$ARCH ~/go/bin/xk6 build $K6_VERSION \
  --output $OUTPUT_NAME \
  --with xk6-output-penelopa=/Users/vitinschiiartiom/GolandProjects/xk6-output-gauge-only

echo "Build completed successfully!"
echo "Binary: $OUTPUT_NAME"

# Make executable
chmod +x $OUTPUT_NAME

echo "You can now use: ./$OUTPUT_NAME run --out penelopa your-test.js" 