#!/bin/bash

echo "=== COMPREHENSIVE TYPE DUPLICATION ANALYSIS ==="
echo

# Create temporary file for analysis
TEMP_FILE="/tmp/type_analysis.txt"
> $TEMP_FILE

echo "Scanning all Go files in pkg/dashboard/ for type declarations..."
echo

# Extract all type declarations with file and line number
find pkg/dashboard/ -name "*.go" -exec grep -Hn "^type " {} \; | \
    sed 's/^pkg\/dashboard\///' | \
    grep -E "type\s+[A-Z][a-zA-Z0-9_]*\s+(struct|interface|func|=)" | \
    while read -r line; do
        file=$(echo "$line" | cut -d: -f1)
        lineno=$(echo "$line" | cut -d: -f2)
        type_line=$(echo "$line" | cut -d: -f3-)
        type_name=$(echo "$type_line" | sed -E 's/type\s+([A-Z][a-zA-Z0-9_]*)\s+.*/\1/')
        echo "$type_name|$file:$lineno|$type_line" >> $TEMP_FILE
    done

# Sort and find duplicates
echo "=== DUPLICATE TYPE ANALYSIS ==="
echo

# Group by type name and show files where each type is declared
sort $TEMP_FILE | cut -d'|' -f1 | uniq -c | while read -r count type_name; do
    if [ $count -gt 1 ]; then
        echo "❌ DUPLICATE: $type_name (found in $count files)"
        grep "^$type_name|" $TEMP_FILE | while IFS='|' read -r name location definition; do
            echo "   - $location: $definition"
        done
        echo
    fi
done

echo "=== ALL TYPE DECLARATIONS BY FILE ==="
echo

# Show all types grouped by file
sort -t'|' -k2 $TEMP_FILE | while IFS='|' read -r type_name location definition; do
    echo "$location: $type_name"
done

# Clean up
rm -f $TEMP_FILE

echo
echo "=== SCAN COMPLETE ==="