#!/bin/bash

# Define target operating systems and architectures
OS=("linux" "darwin" "windows")
ARCH=("amd64" "arm64")

# Remove the build directory if it exists
rm -rf build

# Create build directory if it doesn't exist
mkdir -p build

# Loop through each OS and architecture combination
for os in "${OS[@]}"; do
  for arch in "${ARCH[@]}"; do
    output="build/stream-${os}-${arch}"
    if [ "$os" == "windows" ]; then
      output+=".exe"
    fi
    echo "Building for $os/$arch..."
    GOOS=$os GOARCH=$arch go build -o $output cmd/*.go
  done
done

echo "Build complete. Artifacts are in the build directory."
