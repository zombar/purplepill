#!/bin/bash
# Calculate next semantic version based on conventional commits
# Usage: ./calculate-version.sh [latest_tag]
# Returns: NEW_VERSION (e.g., "1.0.0") and NEW_TAG (e.g., "v1.0.0")

set -e

# Get latest tag from argument or git
LATEST_TAG="${1:-$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")}"

# Check if this is the first release (no tags exist)
if [ "$LATEST_TAG" = "v0.0.0" ]; then
    # First release - start at v1.0.0
    echo "No previous tags found - creating initial release" >&2
    NEW_VERSION="1.0.0"
    NEW_TAG="v1.0.0"
else
    # Strip 'v' prefix if present
    LATEST_VERSION="${LATEST_TAG#v}"

    # Parse current version
    IFS='.' read -r MAJOR MINOR PATCH <<< "$LATEST_VERSION"

    # Get commits since last tag (only if tag exists in git history)
    if git rev-parse "${LATEST_TAG}" >/dev/null 2>&1; then
        COMMITS=$(git log ${LATEST_TAG}..HEAD --pretty=format:"%s")
    else
        # Tag doesn't exist in history, get all commits
        COMMITS=$(git log --pretty=format:"%s")
    fi

    # Determine version bump based on conventional commits
    BREAKING=false
    FEAT=false
    FIX=false

    while IFS= read -r commit; do
        if echo "$commit" | grep -qE "BREAKING CHANGE|!:"; then
            BREAKING=true
        elif echo "$commit" | grep -qE "^feat(\(.+\))?:"; then
            FEAT=true
        elif echo "$commit" | grep -qE "^fix(\(.+\))?:"; then
            FIX=true
        fi
    done <<< "$COMMITS"

    # Bump version
    if [ "$BREAKING" = true ]; then
        MAJOR=$((MAJOR + 1))
        MINOR=0
        PATCH=0
        echo "Version bump: MAJOR (breaking change)" >&2
    elif [ "$FEAT" = true ]; then
        MINOR=$((MINOR + 1))
        PATCH=0
        echo "Version bump: MINOR (new feature)" >&2
    elif [ "$FIX" = true ]; then
        PATCH=$((PATCH + 1))
        echo "Version bump: PATCH (bug fix)" >&2
    else
        PATCH=$((PATCH + 1))
        echo "Version bump: PATCH (default)" >&2
    fi

    NEW_VERSION="$MAJOR.$MINOR.$PATCH"
    NEW_TAG="v$NEW_VERSION"
fi

# Output for GitHub Actions or scripts
echo "version=$NEW_VERSION"
echo "tag=$NEW_TAG"
echo "Next version: $NEW_VERSION (tag: $NEW_TAG)" >&2
