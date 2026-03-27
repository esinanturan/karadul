#!/bin/bash
# Update Homebrew formula with new version and checksums
# Usage: ./update-homebrew.sh v0.1.0

set -e

VERSION=${1#v}  # Remove 'v' prefix if present
REPO="karadul/karadul"

if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 v0.1.0"
    exit 1
fi

echo "Updating Homebrew formula for version $VERSION..."

# Download binaries and calculate checksums
cd /tmp

PLATFORMS=(
    "karadul-darwin-arm64"
    "karadul-darwin-amd64"
    "karadul-linux-arm64"
    "karadul-linux-amd64"
)

SHAS=()
for platform in "${PLATFORMS[@]}"; do
    echo "Downloading $platform..."
    curl -sL -o "$platform" "https://github.com/$REPO/releases/download/v$VERSION/$platform"
    sha=$(sha256sum "$platform" | cut -d' ' -f1)
    SHAS+=("$sha")
    echo "  SHA256: $sha"
done

# Update formula
cd -
TEMPLATE="contrib/homebrew/karadul.rb.template"
OUTPUT="contrib/homebrew/karadul.rb"

cp "$TEMPLATE" "$OUTPUT"

# Replace version and checksums
sed -i.bak "s/VERSION_PLACEHOLDER/$VERSION/g" "$OUTPUT"
sed -i.bak "s/SHA256_DARWIN_ARM64_PLACEHOLDER/${SHAS[0]}/g" "$OUTPUT"
sed -i.bak "s/SHA256_DARWIN_AMD64_PLACEHOLDER/${SHAS[1]}/g" "$OUTPUT"
sed -i.bak "s/SHA256_LINUX_ARM64_PLACEHOLDER/${SHAS[2]}/g" "$OUTPUT"
sed -i.bak "s/SHA256_LINUX_AMD64_PLACEHOLDER/${SHAS[3]}/g" "$OUTPUT"

rm "$OUTPUT.bak"

echo ""
echo "Formula updated: $OUTPUT"
echo ""
echo "To publish to homebrew-tap:"
echo "  1. Clone karadul/homebrew-karadul repo"
echo "  2. Copy $OUTPUT to Formula/karadul.rb"
echo "  3. Commit and push"
