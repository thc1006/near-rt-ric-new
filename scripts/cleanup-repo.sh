#!/bin/bash
# Repository cleanup script for O-RAN Near-RT RIC
# This script removes binaries, coverage files, and other artifacts from git

set -euo pipefail

echo "=========================================="
echo "O-RAN Near-RT RIC Repository Cleanup"
echo "=========================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[✓]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[!]${NC} $1"
}

print_error() {
    echo -e "${RED}[✗]${NC} $1"
}

# Check if we're in a git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    print_error "This script must be run from within a git repository"
    exit 1
fi

# Get repository size before cleanup
REPO_SIZE_BEFORE=$(du -sh .git 2>/dev/null | cut -f1)
echo "Repository size before cleanup: $REPO_SIZE_BEFORE"

# Step 1: Remove binary files from git tracking
echo ""
echo "Step 1: Removing binary files from git tracking..."
echo "-------------------------------------------"

BINARIES_TO_REMOVE=(
    "*.exe"
    "*.out"
    "*.bin"
    "coverage"
    "coverage.*"
    "*.test"
)

for pattern in "${BINARIES_TO_REMOVE[@]}"; do
    if git ls-files "$pattern" --error-unmatch 2>/dev/null; then
        git rm --cached -r $pattern 2>/dev/null || true
        print_status "Removed $pattern from git"
    fi
done

# Step 2: Remove node_modules if tracked
echo ""
echo "Step 2: Checking for tracked node_modules..."
echo "-------------------------------------------"

if [ -d "ui/node_modules" ] && git ls-files --error-unmatch ui/node_modules 2>/dev/null; then
    print_warning "Found tracked node_modules, removing from git..."
    git rm -r --cached ui/node_modules
    print_status "Removed ui/node_modules from git"
fi

# Step 3: Clean up empty directories
echo ""
echo "Step 3: Removing empty directories..."
echo "-------------------------------------------"

EMPTY_DIRS=(
    "oran-sc-dep"
    "oran-sc-dep-shallow"
    "oran-sc-e2mgr"
    "oran-sc-submgr"
    "ric-dep"
)

for dir in "${EMPTY_DIRS[@]}"; do
    if [ -d "$dir" ] && [ -z "$(ls -A $dir)" ]; then
        rmdir "$dir"
        print_status "Removed empty directory: $dir"
    fi
done

# Step 4: Clean build artifacts
echo ""
echo "Step 4: Cleaning build artifacts..."
echo "-------------------------------------------"

# Remove local binary files
rm -f *.exe *.out coverage coverage.* *.test 2>/dev/null || true
print_status "Removed local binary files"

# Clean Go build cache
if command -v go &> /dev/null; then
    go clean -cache -testcache -modcache 2>/dev/null || true
    print_status "Cleaned Go build cache"
fi

# Step 5: Create/update .gitignore if needed
echo ""
echo "Step 5: Verifying .gitignore..."
echo "-------------------------------------------"

if [ -f ".gitignore" ]; then
    print_status ".gitignore exists"
else
    print_error ".gitignore not found! Please create one."
fi

# Step 6: Optimize git repository
echo ""
echo "Step 6: Optimizing git repository..."
echo "-------------------------------------------"

# Run git garbage collection
git gc --aggressive --prune=now
print_status "Ran git garbage collection"

# Repack repository
git repack -a -d -f --depth=250 --window=250
print_status "Repacked repository"

# Clean reflog
git reflog expire --expire=now --all
print_status "Cleaned reflog"

# Step 7: Show statistics
echo ""
echo "=========================================="
echo "Cleanup Statistics"
echo "=========================================="

# Get repository size after cleanup
REPO_SIZE_AFTER=$(du -sh .git 2>/dev/null | cut -f1)
echo "Repository size before: $REPO_SIZE_BEFORE"
echo "Repository size after:  $REPO_SIZE_AFTER"

# Count files
TOTAL_FILES=$(find . -type f -not -path "./.git/*" -not -path "./ui/node_modules/*" | wc -l)
MD_FILES=$(find . -name "*.md" -not -path "./ui/node_modules/*" | wc -l)
GO_FILES=$(find . -name "*.go" | wc -l)

echo ""
echo "File statistics:"
echo "  Total files: $TOTAL_FILES"
echo "  Go files: $GO_FILES"
echo "  Markdown files: $MD_FILES (excluding node_modules)"

# Check for remaining large files
echo ""
echo "Large files still in repository (>1MB):"
find . -type f -size +1M -not -path "./.git/*" -not -path "./ui/node_modules/*" -exec ls -lh {} \; 2>/dev/null | awk '{print $9 ": " $5}'

# Step 8: Suggest next steps
echo ""
echo "=========================================="
echo "Next Steps"
echo "=========================================="
echo "1. Review the changes:"
echo "   git status"
echo ""
echo "2. If everything looks good, commit the changes:"
echo "   git add ."
echo "   git commit -m 'chore: clean repository and remove binaries'"
echo ""
echo "3. Consider running this to remove large files from history (CAUTION: rewrites history):"
echo "   git filter-branch --force --index-filter \\"
echo "     'git rm --cached --ignore-unmatch *.exe *.out coverage' \\"
echo "     --prune-empty --tag-name-filter cat -- --all"
echo ""
echo "4. After history rewrite (if done), force push:"
echo "   git push --force --all"
echo "   git push --force --tags"

print_status "Cleanup complete!"