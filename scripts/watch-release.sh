#!/bin/bash
# GitHub Actions Release Monitor
# Usage: ./scripts/watch-release.sh [tag]

TAG="${1:-v0.1.0-beta.1}"
REPO="karadul/karadul"

echo "📊 Karadul Release Monitor"
echo "=========================="
echo "Tag: $TAG"
echo "Repo: $REPO"
echo ""

# Check if gh CLI is available
if ! command -v gh &> /dev/null; then
    echo "⚠️  GitHub CLI (gh) not found"
    echo "Install: https://cli.github.com/"
    echo ""
    echo "Manual check:"
    echo "  https://github.com/$REPO/actions"
    exit 1
fi

echo "⏳ Checking workflow status..."
echo ""

# Get the latest run for this tag
RUN_ID=$(gh run list --repo "$REPO" --branch "$TAG" --json databaseId,event --jq '.[0].databaseId' 2>/dev/null)

if [ -z "$RUN_ID" ] || [ "$RUN_ID" = "null" ]; then
    echo "❌ No workflow runs found for tag $TAG"
    echo ""
    echo "Possible reasons:"
    echo "  1. Tag was just pushed (wait 30-60 seconds)"
    echo "  2. Actions workflow is disabled"
    echo "  3. Tag format is incorrect"
    echo ""
    echo "Check manually:"
    echo "  https://github.com/$REPO/actions"
    exit 1
fi

echo "✅ Found workflow run: $RUN_ID"
echo ""

# Display run status
gh run view "$RUN_ID" --repo "$REPO"

echo ""
echo "📊 Workflow URL:"
echo "  https://github.com/$REPO/actions/runs/$RUN_ID"
