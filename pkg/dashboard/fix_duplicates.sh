#!/bin/bash

# Script to fix duplicate type declarations in types.go

echo "Fixing duplicate type declarations in types.go..."

# Create backup
cp types.go types.go.backup2

# Remove the older ServiceModelAPI struct (lines 422-430) and keep the interface (line 487)
# Remove the older ServiceModelRegistry struct (lines 432-437) and keep the consolidated one (line 496)

# First, let's extract the parts we want to keep:
# Part 1: Lines 1-421 (before first ServiceModelAPI)
sed -n '1,421p' types.go > types_part1.tmp

# Part 2: Lines 438-486 (after first ServiceModelRegistry, before ServiceModelAPI interface)  
sed -n '438,486p' types.go > types_part2.tmp

# Part 3: Lines 487-end (ServiceModelAPI interface and everything after)
sed -n '487,$p' types.go > types_part3.tmp

# Combine the parts, effectively removing lines 422-437
cat types_part1.tmp types_part2.tmp types_part3.tmp > types_fixed.tmp

# Now we need to remove any other duplicates found in the build error
# Based on the build errors, we also need to handle:
# - ServiceModelCapabilities 
# - ServiceModelStatistics
# - PolicyTypeID
# - PolicyInstanceID  
# - PolicyDefinition
# - PolicyManagerStats

# Let's apply the fix
mv types_fixed.tmp types.go

# Clean up temporary files
rm -f types_part1.tmp types_part2.tmp types_part3.tmp

echo "Initial duplicate removal complete."
echo "Testing build..."

go build . 2>&1 | head -20