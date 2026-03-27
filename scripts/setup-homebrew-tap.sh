#!/bin/bash
# Setup Homebrew Tap for Karadul
# This script creates a separate homebrew-karadul repository

set -e

REPO_OWNER="karadul"
TAP_REPO="homebrew-karadul"
FORMULA_DIR="Formula"

echo "🍺 Karadul Homebrew Tap Setup"
echo "=============================="
echo ""

# Check if gh CLI is available
if ! command -v gh &> /dev/null; then
    echo "❌ GitHub CLI (gh) is required"
    echo "Install: https://cli.github.com/"
    exit 1
fi

# Check if logged in to GitHub
if ! gh auth status &> /dev/null; then
    echo "❌ Not logged in to GitHub"
    echo "Run: gh auth login"
    exit 1
fi

echo "✅ GitHub CLI authenticated"
echo ""

# Check if tap repo exists
echo "Checking if tap repository exists..."
if gh repo view "$REPO_OWNER/$TAP_REPO" &> /dev/null; then
    echo "✅ Tap repository already exists: $REPO_OWNER/$TAP_REPO"
else
    echo "Creating tap repository..."
    gh repo create "$TAP_REPO" \
        --public \
        --description "Homebrew tap for Karadul mesh VPN" \
        --homepage "https://github.com/karadul/karadul"
    echo "✅ Created: $REPO_OWNER/$TAP_REPO"
fi

# Clone tap repository
TEMP_DIR=$(mktemp -d)
echo ""
echo "Cloning tap repository to $TEMP_DIR..."
gh repo clone "$REPO_OWNER/$TAP_REPO" "$TEMP_DIR"

# Create Formula directory
mkdir -p "$TEMP_DIR/$FORMULA_DIR"

# Copy formula template
echo ""
echo "Creating initial formula..."
cp "contrib/homebrew/karadul.rb.template" "$TEMP_DIR/$FORMULA_DIR/karadul.rb"

# Get latest release info
echo ""
echo "Fetching latest release info..."
LATEST_RELEASE=$(gh release list --repo "karadul/karadul" --limit 1 --json tagName --jq '.[0].tagName')

if [ -n "$LATEST_RELEASE" ]; then
    echo "Latest release: $LATEST_RELEASE"

    # Update VERSION in formula
    VERSION="${LATEST_RELEASE#v}"
    sed -i.bak "s/VERSION_PLACEHOLDER/$VERSION/g" "$TEMP_DIR/$FORMULA_DIR/karadul.rb"
    rm "$TEMP_DIR/$FORMULA_DIR/karadul.rb.bak"

    # Download binaries and calculate SHA256
    echo ""
    echo "Downloading binaries and calculating checksums..."

    PLATFORMS=(
        "karadul-darwin-arm64:SHA256_DARWIN_ARM64_PLACEHOLDER"
        "karadul-darwin-amd64:SHA256_DARWIN_AMD64_PLACEHOLDER"
        "karadul-linux-arm64:SHA256_LINUX_ARM64_PLACEHOLDER"
        "karadul-linux-amd64:SHA256_LINUX_AMD64_PLACEHOLDER"
    )

    cd "$TEMP_DIR"

    for platform_info in "${PLATFORMS[@]}"; do
        IFS=':' read -r binary placeholder <<< "$platform_info"

        echo -n "  Downloading $binary... "
        if gh release download "$LATEST_RELEASE" --repo "karadul/karadul" -p "$binary" -O "$binary" 2>/dev/null; then
            SHA=$(sha256sum "$binary" | cut -d' ' -f1)
            sed -i.bak "s/$placeholder/$SHA/g" "$FORMULA_DIR/karadul.rb"
            rm "$FORMULA_DIR/karadul.rb.bak"
            rm "$binary"
            echo "✓"
        else
            echo "✗ (not found in release)"
        fi
    done
fi

# Create README for tap
cat > "$TEMP_DIR/README.md" << 'EOF'
# Karadul Homebrew Tap

Homebrew tap for [Karadul](https://github.com/karadul/karadul) - Self-hosted mesh VPN.

## Installation

```bash
# Add this tap
brew tap karadul/karadul

# Install Karadul
brew install karadul
```

## Usage

```bash
# Start coordination server
karadul server --addr=:8080

# Join mesh as a node
karadul up --server=https://your-server:8080 --auth-key=<key>
```

See [Karadul documentation](https://github.com/karadul/karadul#readme) for more details.

## Updates

```bash
brew update
brew upgrade karadul
```

## Uninstall

```bash
brew uninstall karadul
brew untap karadul/karadul
```
EOF

# Commit and push
echo ""
echo "Committing changes..."
cd "$TEMP_DIR"
git add -A
git commit -m "karadul: add formula v${VERSION:-0.1.0}" || echo "No changes to commit"

echo ""
echo "Pushing to GitHub..."
git push origin main || echo "Nothing to push"

# Cleanup
cd - > /dev/null
rm -rf "$TEMP_DIR"

echo ""
echo "✅ Homebrew tap setup complete!"
echo ""
echo "Users can now install with:"
echo "  brew tap $REPO_OWNER/$TAP_REPO"
echo "  brew install karadul"
echo ""
echo "Repository: https://github.com/$REPO_OWNER/$TAP_REPO"
