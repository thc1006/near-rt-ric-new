# Final Type Consolidation Script
# This completes the type consolidation by removing remaining duplicates

Write-Host "=== Final Type Consolidation ===" -ForegroundColor Cyan

# List of files and types to clean up
$cleanupMap = @{
    "pkg\dashboard\a1_models.go" = @(
        "PolicyTypeID",
        "PolicyInstanceID", 
        "PolicyInstance",
        "PolicyConflict",
        "PolicyDistributionStatus"
    )
    "pkg\dashboard\policy_manager.go" = @(
        "XAppClient",
        "PolicyDefinition",
        "PolicyManagerStats"
    )
    "pkg\dashboard\smo_components.go" = @(
        "PolicyDefinition",
        "PolicyManagerStats"
    )
}

# Function to remove type declarations from a file
function Remove-TypeDeclaration {
    param(
        [string]$FilePath,
        [string]$TypeName
    )
    
    Write-Host "  Removing $TypeName from $(Split-Path -Leaf $FilePath)..." -ForegroundColor Yellow
    
    $content = Get-Content $FilePath
    $newContent = @()
    $inType = $false
    $braceCount = 0
    
    for ($i = 0; $i -lt $content.Count; $i++) {
        $line = $content[$i]
        
        # Check for type declaration
        if ($line -match "^type\s+$TypeName\s+(struct|interface|string|int)") {
            $inType = $true
            
            # Handle simple type aliases (string, int, etc.)
            if ($line -match "^type\s+$TypeName\s+(string|int|int32|int64|float32|float64|bool)") {
                # Single line type alias - skip it
                $newContent += "// $TypeName is now defined in types.go to avoid redeclaration"
                $inType = $false
                continue
            }
            
            # Handle struct/interface
            if ($line -match "\{") { 
                $braceCount = 1
                # Check if it's a single-line struct
                if ($line -match "\{.*\}") {
                    $newContent += "// $TypeName is now defined in types.go to avoid redeclaration"
                    $inType = $false
                    continue
                }
            } else {
                # Struct opening brace might be on next line
                $braceCount = 0
            }
            
            # Add comment instead of type declaration
            $newContent += "// $TypeName is now defined in types.go to avoid redeclaration"
            continue
        }
        
        if ($inType) {
            # Count braces
            $openBraces = ([regex]::Matches($line, '\{').Count)
            $closeBraces = ([regex]::Matches($line, '\}').Count)
            $braceCount += $openBraces - $closeBraces
            
            # Check if type definition is complete
            if ($braceCount -eq 0) {
                $inType = $false
            }
            
            # Skip lines that are part of the type being removed
            continue
        }
        
        # Add all other lines
        $newContent += $line
    }
    
    Set-Content -Path $FilePath -Value $newContent
}

# Process each file
foreach ($file in $cleanupMap.Keys) {
    if (Test-Path $file) {
        Write-Host "Processing: $file" -ForegroundColor Green
        
        foreach ($typeName in $cleanupMap[$file]) {
            Remove-TypeDeclaration -FilePath $file -TypeName $typeName
        }
    } else {
        Write-Host "File not found: $file" -ForegroundColor Red
    }
}

# Also check for duplicates within types.go itself
Write-Host "`nChecking for duplicates within types.go..." -ForegroundColor Cyan

$typesFile = "pkg\dashboard\types.go"
$content = Get-Content $typesFile

# Find all type declarations and their line numbers
$typeDeclarations = @{}
for ($i = 0; $i -lt $content.Count; $i++) {
    $line = $content[$i]
    if ($line -match '^type\s+([A-Za-z][A-Za-z0-9]*)\s+(struct|interface|string|int)') {
        $typeName = $matches[1]
        if ($typeDeclarations.ContainsKey($typeName)) {
            Write-Host "  Found duplicate: $typeName at line $($i + 1) (first at $($typeDeclarations[$typeName] + 1))" -ForegroundColor Yellow
        } else {
            $typeDeclarations[$typeName] = $i
        }
    }
}

Write-Host "`n=== Testing Build ===" -ForegroundColor Cyan
$errors = go build ./cmd/dashboard-api 2>&1 | Out-String

if ($errors -match "redeclared") {
    Write-Host "Still have redeclaration errors:" -ForegroundColor Red
    $errors -split "`n" | Where-Object { $_ -match "redeclared" } | ForEach-Object {
        Write-Host "  $_" -ForegroundColor Red
    }
} else {
    Write-Host "SUCCESS! No redeclaration errors found." -ForegroundColor Green
}

Write-Host "`n=== Final Type Consolidation Complete ===" -ForegroundColor Cyan