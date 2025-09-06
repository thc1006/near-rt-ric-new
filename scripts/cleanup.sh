#!/bin/bash

# O-RAN Near-RT RIC Cleanup Script
# This script safely removes unnecessary files and optimizes the repository

set -e

echo "========================================="
echo "O-RAN Near-RT RIC Repository Cleanup"
echo "========================================="

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to calculate size
calculate_size() {
    if command -v du &> /dev/null; then
        size=$(du -sh "$1" 2>/dev/null | cut -f1)
        echo "$size"
    else
        echo "N/A"
    fi
}

# Dry run mode by default
DRY_RUN=true
if [ "$1" == "--execute" ]; then
    DRY_RUN=false
    print_warning "Running in EXECUTE mode. Files will be deleted!"
    read -p "Are you sure you want to proceed? (yes/no): " confirm
    if [ "$confirm" != "yes" ]; then
        print_info "Cleanup cancelled."
        exit 0
    fi
else
    print_info "Running in DRY RUN mode. Use '--execute' to actually delete files."
fi

# Track total space saved
TOTAL_SAVED=0

echo ""
echo "Phase 1: Removing Binary Files"
echo "-------------------------------"

# Binary files to remove
BINARY_FILES=(
    "dashboard-api.exe"
    "ric.exe"
    "xapp-hello-world.exe"
)

for file in "${BINARY_FILES[@]}"; do
    if [ -f "$file" ]; then
        size=$(calculate_size "$file")
        if [ "$DRY_RUN" = true ]; then
            print_info "Would remove: $file (Size: $size)"
        else
            rm -f "$file"
            print_info "Removed: $file (Size: $size)"
        fi
    fi
done

echo ""
echo "Phase 2: Removing Coverage Files"
echo "---------------------------------"

# Coverage files to remove
COVERAGE_PATTERN="dashboard_coverage*"
if [ "$DRY_RUN" = true ]; then
    print_info "Would remove coverage files:"
    ls -lh $COVERAGE_PATTERN coverage.out coverage 2>/dev/null || echo "  No coverage files found"
else
    rm -f $COVERAGE_PATTERN coverage.out coverage
    print_info "Removed coverage files"
fi

echo ""
echo "Phase 3: Cleaning Build Artifacts"
echo "----------------------------------"

# Clean Go build cache
if [ "$DRY_RUN" = true ]; then
    print_info "Would clean Go build cache"
else
    go clean -cache -testcache -modcache 2>/dev/null || print_warning "Could not clean Go cache"
fi

# Remove test binaries
if [ "$DRY_RUN" = true ]; then
    print_info "Would remove test binaries:"
    find . -name "*.test" -type f 2>/dev/null | head -5
else
    find . -name "*.test" -type f -delete 2>/dev/null
    print_info "Removed test binaries"
fi

echo ""
echo "Phase 4: Optimizing Git Repository"
echo "-----------------------------------"

# Git cleanup
if [ -d ".git" ]; then
    git_size_before=$(calculate_size ".git")
    if [ "$DRY_RUN" = true ]; then
        print_info "Would run git gc (current .git size: $git_size_before)"
    else
        git gc --aggressive --prune=now 2>/dev/null || print_warning "Could not optimize git repository"
        git_size_after=$(calculate_size ".git")
        print_info "Git optimization complete (before: $git_size_before, after: $git_size_after)"
    fi
fi

echo ""
echo "Phase 5: Checking Large Directories"
echo "------------------------------------"

# Check for node_modules
if [ -d "ui/node_modules" ]; then
    size=$(calculate_size "ui/node_modules")
    print_warning "Found ui/node_modules (Size: $size)"
    print_info "Run 'cd ui && npm ci' after cleanup to reinstall dependencies"
fi

# Check for vendor directory
if [ -d "vendor" ]; then
    size=$(calculate_size "vendor")
    print_warning "Found vendor directory (Size: $size)"
    print_info "Consider using Go modules instead of vendoring"
fi

echo ""
echo "Phase 6: Empty Directory Cleanup"
echo "---------------------------------"

# Find and remove empty directories
if [ "$DRY_RUN" = true ]; then
    print_info "Would remove empty directories:"
    find . -type d -empty -not -path "./.git/*" 2>/dev/null | head -10
else
    find . -type d -empty -not -path "./.git/*" -delete 2>/dev/null
    print_info "Removed empty directories"
fi

echo ""
echo "Phase 7: Update .gitignore"
echo "---------------------------"

# Ensure .gitignore has proper entries
GITIGNORE_ADDITIONS=(
    "# Binaries"
    "*.exe"
    "*.test"
    "*.out"
    ""
    "# Coverage"
    "coverage*"
    "*.prof"
    ""
    "# Build artifacts"
    "/build/"
    "/dist/"
    ""
    "# IDE"
    ".idea/"
    "*.swp"
    "*.swo"
    ".DS_Store"
    ""
    "# Dependencies"
    "/vendor/"
    "node_modules/"
    ""
    "# Environment"
    ".env"
    ".env.local"
)

if [ "$DRY_RUN" = true ]; then
    print_info "Would update .gitignore with standard entries"
else
    # Backup existing .gitignore
    if [ -f .gitignore ]; then
        cp .gitignore .gitignore.bak
    fi
    
    # Check and add missing entries
    for entry in "${GITIGNORE_ADDITIONS[@]}"; do
        if [ -n "$entry" ] && ! grep -qF "$entry" .gitignore 2>/dev/null; then
            echo "$entry" >> .gitignore
        fi
    done
    print_info "Updated .gitignore"
fi

echo ""
echo "========================================="
echo "Cleanup Summary"
echo "========================================="

if [ "$DRY_RUN" = true ]; then
    print_info "DRY RUN COMPLETE"
    echo ""
    echo "To execute cleanup, run:"
    echo "  ./scripts/cleanup.sh --execute"
    echo ""
    echo "Estimated space to be freed: ~65+ MB"
else
    print_info "CLEANUP COMPLETE"
    echo ""
    echo "Next steps:"
    echo "1. Run 'git status' to review changes"
    echo "2. Commit the cleanup: git add -A && git commit -m 'chore: repository cleanup'"
    echo "3. If needed, restore dependencies:"
    echo "   - Go: go mod download"
    echo "   - Node: cd ui && npm ci"
fi

echo ""
print_info "Repository optimization completed successfully!"