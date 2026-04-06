#!/bin/bash
# Build AniClew binaries for all platforms

VERSION=${1:-"dev"}
OUTPUT_DIR="dist"
MODULE="github.com/aniclew/aniclew"

echo "Building AniClew $VERSION..."
mkdir -p $OUTPUT_DIR

# Build frontend first
echo "Building frontend..."
cd web && npm run build && cd ..
rm -f internal/server/webdist/assets/index-*.js internal/server/webdist/assets/index-*.css
cp -r web/dist/* internal/server/webdist/

# Build for each platform
platforms=(
  "windows/amd64:.exe"
  "darwin/amd64:"
  "darwin/arm64:"
  "linux/amd64:"
  "linux/arm64:"
)

for platform in "${platforms[@]}"; do
  IFS=':' read -r os_arch ext <<< "$platform"
  IFS='/' read -r os arch <<< "$os_arch"

  output="$OUTPUT_DIR/aniclew-${VERSION}-${os}-${arch}${ext}"
  echo "  Building $os/$arch → $output"

  GOOS=$os GOARCH=$arch go build -ldflags "-s -w -X main.version=$VERSION" -o "$output" ./cmd/proxy

  if [ $? -eq 0 ]; then
    echo "    ✓ $(du -h "$output" | cut -f1)"
  else
    echo "    ✗ Failed"
  fi
done

# Create checksums
echo "Creating checksums..."
cd $OUTPUT_DIR
sha256sum aniclew-* > checksums.txt 2>/dev/null || shasum -a 256 aniclew-* > checksums.txt
cd ..

echo ""
echo "Done! Binaries in $OUTPUT_DIR/"
ls -lh $OUTPUT_DIR/
