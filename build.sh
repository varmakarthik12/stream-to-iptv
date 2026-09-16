#!/bin/bash
set -e

echo "Building React Web UI bundle..."
npm --prefix web install
npm --prefix web run build

# Define target operating systems and architectures
OS=("linux" "darwin" "windows")
ARCH=("amd64" "arm64")

# Remove and recreate build directory
rm -rf build
mkdir -p build

# Loop through each OS and architecture combination
for os in "${OS[@]}"; do
  for arch in "${ARCH[@]}"; do
    # Skip Windows ARM64 if not needed
    if [ "$os" == "windows" ] && [ "$arch" == "arm64" ]; then
      continue
    fi

    output="build/stream-${os}-${arch}"
    if [ "$os" == "windows" ]; then
      output+=".exe"
    fi
    echo "Building for $os/$arch..."
    CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -ldflags="-w -s" -o "$output" ./cmd
  done
done

echo "Build complete. Multi-platform artifacts with embedded UI are in the build directory."
