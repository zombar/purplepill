#!/bin/bash
# Tests for calculate-version.sh
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CALCULATE_VERSION="$SCRIPT_DIR/calculate-version.sh"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Test helper function
test_version_calculation() {
    local test_name="$1"
    local latest_tag="$2"
    local commits="$3"
    local expected_version="$4"
    local expected_tag="$5"

    TESTS_RUN=$((TESTS_RUN + 1))

    echo -e "\n${YELLOW}Test $TESTS_RUN: $test_name${NC}"
    echo "  Latest tag: $latest_tag"
    echo "  Commits: $commits"

    # Create a temporary git repo for testing
    TEST_DIR=$(mktemp -d)
    cd "$TEST_DIR"
    git init -q
    git config user.email "test@example.com"
    git config user.name "Test User"

    # Create initial commit
    echo "initial" > file.txt
    git add file.txt
    git commit -q -m "Initial commit"

    # Create tag if not first release
    if [ "$latest_tag" != "v0.0.0" ]; then
        git tag "$latest_tag"
    fi

    # Add commits
    if [ -n "$commits" ]; then
        IFS='|' read -ra COMMIT_ARRAY <<< "$commits"
        for commit_msg in "${COMMIT_ARRAY[@]}"; do
            echo "change" >> file.txt
            git add file.txt
            git commit -q -m "$commit_msg"
        done
    fi

    # Run version calculation
    OUTPUT=$("$CALCULATE_VERSION" "$latest_tag" 2>/dev/null)
    ACTUAL_VERSION=$(echo "$OUTPUT" | grep "^version=" | cut -d= -f2)
    ACTUAL_TAG=$(echo "$OUTPUT" | grep "^tag=" | cut -d= -f2)

    # Clean up
    cd /
    rm -rf "$TEST_DIR"

    # Check results
    if [ "$ACTUAL_VERSION" = "$expected_version" ] && [ "$ACTUAL_TAG" = "$expected_tag" ]; then
        echo -e "  ${GREEN}✓ PASS${NC} - Got $ACTUAL_TAG"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo -e "  ${RED}✗ FAIL${NC}"
        echo "    Expected: $expected_tag ($expected_version)"
        echo "    Got:      $ACTUAL_TAG ($ACTUAL_VERSION)"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

echo "================================"
echo "Version Calculation Tests"
echo "================================"

# Test 1: First release (no previous tags)
test_version_calculation \
    "First release" \
    "v0.0.0" \
    "chore: initial setup" \
    "1.0.0" \
    "v1.0.0"

# Test 2: Patch bump (fix commit)
test_version_calculation \
    "Patch bump with fix commit" \
    "v1.0.0" \
    "fix: resolve memory leak" \
    "1.0.1" \
    "v1.0.1"

# Test 3: Minor bump (feat commit)
test_version_calculation \
    "Minor bump with feat commit" \
    "v1.0.0" \
    "feat: add dark mode support" \
    "1.1.0" \
    "v1.1.0"

# Test 4: Major bump (breaking change)
test_version_calculation \
    "Major bump with BREAKING CHANGE" \
    "v1.5.3" \
    "feat: redesign API|BREAKING CHANGE: API endpoints changed" \
    "2.0.0" \
    "v2.0.0"

# Test 5: Major bump with exclamation mark syntax
test_version_calculation \
    "Major bump with ! syntax" \
    "v1.5.3" \
    "feat!: redesign API endpoints" \
    "2.0.0" \
    "v2.0.0"

# Test 6: Multiple commits with feat taking precedence over fix
test_version_calculation \
    "Multiple commits (feat + fix)" \
    "v1.2.0" \
    "fix: resolve race condition|feat: add export functionality|docs: update README" \
    "1.3.0" \
    "v1.3.0"

# Test 7: Breaking change takes precedence over everything
test_version_calculation \
    "Multiple commits with BREAKING CHANGE" \
    "v2.1.5" \
    "fix: resolve bug|feat: add feature|feat!: breaking change" \
    "3.0.0" \
    "v3.0.0"

# Test 8: No conventional commits - default to patch
test_version_calculation \
    "No conventional commits (default patch)" \
    "v1.0.0" \
    "Update documentation|Refactor code|Fix typo" \
    "1.0.1" \
    "v1.0.1"

# Test 9: Feat with scope
test_version_calculation \
    "Feat with scope" \
    "v1.0.0" \
    "feat(controller): add rate limiting" \
    "1.1.0" \
    "v1.1.0"

# Test 10: Fix with scope
test_version_calculation \
    "Fix with scope" \
    "v1.2.3" \
    "fix(web): resolve rendering issue" \
    "1.2.4" \
    "v1.2.4"

# Test 11: Multiple features - still minor bump
test_version_calculation \
    "Multiple features" \
    "v1.0.0" \
    "feat: add feature A|feat: add feature B|feat: add feature C" \
    "1.1.0" \
    "v1.1.0"

# Test 12: Complex version number
test_version_calculation \
    "Complex version with fix" \
    "v10.25.99" \
    "fix: critical security patch" \
    "10.25.100" \
    "v10.25.100"

# Print summary
echo ""
echo "================================"
echo "Test Summary"
echo "================================"
echo "Tests run:    $TESTS_RUN"
echo -e "Tests passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Tests failed: ${RED}$TESTS_FAILED${NC}"
echo "================================"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
fi
