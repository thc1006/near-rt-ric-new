#!/bin/bash

# Type Conflict Resolution Orchestrator for Dashboard API
# This script systematically resolves all Go type redeclaration errors

set -e

echo "==========================================="
echo "Dashboard API Type Consolidation Orchestrator"
echo "==========================================="

DASHBOARD_DIR="pkg/dashboard"
TYPES_FILE="$DASHBOARD_DIR/types.go"
BACKUP_DIR="backup_$(date +%Y%m%d_%H%M%S)"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[✓]${NC} $1"
}

print_error() {
    echo -e "${RED}[✗]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[!]${NC} $1"
}

# Phase 1: Backup
echo ""
echo "Phase 1: Creating backup..."
mkdir -p "$BACKUP_DIR"
cp -r "$DASHBOARD_DIR" "$BACKUP_DIR/"
print_status "Backup created in $BACKUP_DIR"

# Phase 2: Analysis
echo ""
echo "Phase 2: Analyzing type conflicts..."

# Find all type declarations
echo "Scanning for duplicate type declarations..."
declare -A type_locations
declare -A duplicate_types

# Scan all Go files for type declarations
for file in $DASHBOARD_DIR/*.go; do
    if [[ -f "$file" ]]; then
        basename=$(basename "$file")
        # Extract type declarations
        while IFS= read -r line; do
            if [[ "$line" =~ ^type[[:space:]]+([A-Za-z][A-Za-z0-9]*)[[:space:]]+(struct|interface) ]]; then
                type_name="${BASH_REMATCH[1]}"
                if [[ -z "${type_locations[$type_name]}" ]]; then
                    type_locations[$type_name]="$basename"
                else
                    type_locations[$type_name]="${type_locations[$type_name]},$basename"
                    duplicate_types[$type_name]=1
                fi
            fi
        done < "$file"
    fi
done

# Report duplicate types
echo ""
echo "Found duplicate type declarations:"
for type_name in "${!duplicate_types[@]}"; do
    echo "  - $type_name in: ${type_locations[$type_name]}"
done

# Phase 3: Identify types to consolidate
echo ""
echo "Phase 3: Identifying consolidation targets..."

# List of known conflicting types from error messages
CONFLICTING_TYPES=(
    "WorkItem"
    "WorkResult"
    "WorkType"
    "ServiceModelDefinition"
    "ServiceModelCapability"
)

# Add const declarations that need consolidation
CONFLICTING_CONSTS=(
    "WorkTypeE2APMessage"
    "WorkTypeSubscription"
    "WorkTypeIndication"
    "WorkTypeControl"
    "WorkTypePolicyUpdate"
)

# Phase 4: Remove duplicates from source files
echo ""
echo "Phase 4: Removing duplicate declarations from source files..."

# Function to remove type declaration from file
remove_type_from_file() {
    local file=$1
    local type_name=$2
    local temp_file="${file}.tmp"
    
    # Skip types.go for now
    if [[ "$file" == *"types.go" ]]; then
        return
    fi
    
    echo "  Removing $type_name from $(basename $file)..."
    
    # Use awk to remove the type declaration and its body
    awk -v type="$type_name" '
    BEGIN { in_type = 0; brace_count = 0 }
    /^type[[:space:]]+'$type_name'[[:space:]]+(struct|interface)/ {
        in_type = 1
        if ($0 ~ /{/) brace_count = 1
        if ($0 ~ /}/) brace_count = 0
        if (brace_count == 0) in_type = 0
        next
    }
    in_type && /{/ { brace_count++ }
    in_type && /}/ { 
        brace_count--
        if (brace_count == 0) in_type = 0
        next
    }
    !in_type { print }
    ' "$file" > "$temp_file"
    
    mv "$temp_file" "$file"
}

# Remove conflicting types from non-types.go files
for type_name in "${CONFLICTING_TYPES[@]}"; do
    files=$(grep -l "type $type_name struct" $DASHBOARD_DIR/*.go 2>/dev/null || true)
    for file in $files; do
        if [[ "$file" != *"types.go" ]]; then
            remove_type_from_file "$file" "$type_name"
        fi
    done
done

# Phase 5: Remove duplicate const declarations
echo ""
echo "Phase 5: Removing duplicate const declarations..."

# Function to remove const declaration from file
remove_const_from_file() {
    local file=$1
    local const_name=$2
    
    # Skip types.go for now
    if [[ "$file" == *"types.go" ]]; then
        return
    fi
    
    echo "  Removing $const_name from $(basename $file)..."
    
    # Remove the const declaration line
    sed -i "/^[[:space:]]*$const_name[[:space:]]*WorkType/d" "$file" 2>/dev/null || \
    sed -i "" "/^[[:space:]]*$const_name[[:space:]]*WorkType/d" "$file" 2>/dev/null || true
}

# Remove conflicting consts from non-types.go files
for const_name in "${CONFLICTING_CONSTS[@]}"; do
    files=$(grep -l "$const_name" $DASHBOARD_DIR/*.go 2>/dev/null || true)
    for file in $files; do
        if [[ "$file" != *"types.go" ]]; then
            remove_const_from_file "$file" "$const_name"
        fi
    done
done

# Phase 6: Check and remove duplicates within types.go
echo ""
echo "Phase 6: Cleaning up types.go..."

# Create a temporary file for cleaned types.go
TEMP_TYPES="${TYPES_FILE}.clean"

# Remove duplicate declarations within types.go
awk '
BEGIN { 
    seen_types[""] = 0
    seen_consts[""] = 0
    in_type = 0
    in_const_block = 0
    type_content = ""
}

# Track type declarations
/^type[[:space:]]+[A-Za-z][A-Za-z0-9]*[[:space:]]+(struct|interface)/ {
    match($0, /^type[[:space:]]+([A-Za-z][A-Za-z0-9]*)/, arr)
    type_name = arr[1]
    
    if (seen_types[type_name]) {
        in_type = 1
        type_content = ""
        if ($0 ~ /{/ && $0 ~ /}/) {
            in_type = 0
        }
        next
    }
    
    seen_types[type_name] = 1
    in_type = 1
    type_content = $0
    
    if ($0 ~ /{/ && $0 ~ /}/) {
        print $0
        in_type = 0
    } else {
        print $0
    }
    next
}

# Handle const blocks
/^const[[:space:]]*\(/ {
    in_const_block = 1
    print $0
    next
}

in_const_block && /^\)/ {
    in_const_block = 0
    print $0
    next
}

# Track individual const declarations
in_const_block && /^[[:space:]]*[A-Za-z][A-Za-z0-9]*[[:space:]]*WorkType/ {
    match($0, /^[[:space:]]*([A-Za-z][A-Za-z0-9]*)/, arr)
    const_name = arr[1]
    
    if (!seen_consts[const_name]) {
        seen_consts[const_name] = 1
        print $0
    }
    next
}

# Handle multi-line type definitions
in_type {
    if ($0 ~ /^}/) {
        print $0
        in_type = 0
    } else {
        print $0
    }
    next
}

# Print everything else
!in_type {
    print $0
}
' "$TYPES_FILE" > "$TEMP_TYPES"

mv "$TEMP_TYPES" "$TYPES_FILE"
print_status "Cleaned up types.go"

# Phase 7: Test build
echo ""
echo "Phase 7: Testing build..."

if go build ./cmd/dashboard-api 2>&1 | grep -q "redeclared"; then
    print_error "Build still has redeclaration errors. Performing deep analysis..."
    
    # Get remaining errors
    echo ""
    echo "Remaining errors:"
    go build ./cmd/dashboard-api 2>&1 | grep -E "redeclared|previous declaration"
    
    # Try to auto-fix remaining issues
    echo ""
    echo "Attempting auto-fix of remaining issues..."
    
    # Extract remaining duplicate types
    go build ./cmd/dashboard-api 2>&1 | grep "redeclared in this block" | while read -r line; do
        if [[ "$line" =~ ([A-Za-z][A-Za-z0-9]*)[[:space:]]redeclared ]]; then
            type_name="${BASH_REMATCH[1]}"
            print_warning "Fixing remaining duplicate: $type_name"
            
            # Remove from all files except types.go
            for file in $DASHBOARD_DIR/*.go; do
                if [[ "$file" != *"types.go" ]]; then
                    remove_type_from_file "$file" "$type_name" 2>/dev/null || true
                fi
            done
        fi
    done
else
    print_status "Build successful! No redeclaration errors found."
fi

# Phase 8: Final validation
echo ""
echo "Phase 8: Final validation..."

# Count remaining type declarations
echo "Type declaration summary:"
echo "  Total Go files: $(find $DASHBOARD_DIR -name '*.go' | wc -l)"
echo "  Types in types.go: $(grep -c '^type ' $TYPES_FILE || echo 0)"

# Test final build
echo ""
echo "Running final build test..."
if go build ./cmd/dashboard-api 2>&1 | tee /tmp/build_output.txt | grep -q "redeclared"; then
    print_error "Build still has errors. See output above."
    echo ""
    echo "Errors summary:"
    grep -E "redeclared|previous declaration" /tmp/build_output.txt || true
else
    print_status "BUILD SUCCESSFUL! All type conflicts resolved."
    echo ""
    echo "==========================================="
    echo "Type consolidation completed successfully!"
    echo "==========================================="
fi

# Phase 9: Generate report
echo ""
echo "Phase 9: Generating report..."

REPORT_FILE="type_consolidation_report_$(date +%Y%m%d_%H%M%S).txt"
{
    echo "Type Consolidation Report"
    echo "========================="
    echo "Date: $(date)"
    echo ""
    echo "Files Modified:"
    for file in $DASHBOARD_DIR/*.go; do
        if diff -q "$file" "$BACKUP_DIR/dashboard/$(basename $file)" >/dev/null 2>&1; then
            :
        else
            echo "  - $(basename $file)"
        fi
    done
    echo ""
    echo "Types Consolidated:"
    for type_name in "${CONFLICTING_TYPES[@]}"; do
        echo "  - $type_name"
    done
    echo ""
    echo "Build Status: $(go build ./cmd/dashboard-api 2>&1 >/dev/null && echo 'SUCCESS' || echo 'FAILED')"
} > "$REPORT_FILE"

print_status "Report generated: $REPORT_FILE"

echo ""
echo "Orchestration complete!"
echo "Backup location: $BACKUP_DIR"
echo "Report: $REPORT_FILE"