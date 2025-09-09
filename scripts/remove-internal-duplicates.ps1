# Remove duplicate type declarations within types.go itself

$typesFile = "pkg\dashboard\types.go"
Write-Host "Removing internal duplicates from types.go..." -ForegroundColor Cyan

# Read the file
$content = Get-Content $typesFile

# Track seen types and their first occurrence
$seenTypes = @{}
$outputLines = @()
$inType = $false
$currentType = ""
$braceCount = 0
$skipUntilBrace = $false

for ($i = 0; $i -lt $content.Count; $i++) {
    $line = $content[$i]
    
    # Check for type declaration
    if ($line -match '^type\s+([A-Za-z][A-Za-z0-9]*)\s+(struct|interface|string|int|float)') {
        $typeName = $matches[1]
        
        if ($seenTypes.ContainsKey($typeName)) {
            Write-Host "  Removing duplicate: $typeName at line $($i + 1)" -ForegroundColor Yellow
            
            # Skip this duplicate type
            $inType = $true
            $currentType = $typeName
            
            # Check if it's a simple type alias
            if ($line -match '^type\s+\w+\s+(string|int|int32|int64|float32|float64|bool|interface\{\})$') {
                $inType = $false
                continue
            }
            
            # Count braces on the same line
            $openBraces = ([regex]::Matches($line, '\{').Count)
            $closeBraces = ([regex]::Matches($line, '\}').Count)
            $braceCount = $openBraces - $closeBraces
            
            # Check if it's a single-line declaration
            if ($braceCount -eq 0 -and $line -match '\{.*\}') {
                $inType = $false
            }
            continue
        } else {
            $seenTypes[$typeName] = $i
            $outputLines += $line
            
            # Check if it's a simple type alias
            if ($line -match '^type\s+\w+\s+(string|int|int32|int64|float32|float64|bool|interface\{\})$') {
                continue
            }
            
            # Check if it's a single-line declaration
            if ($line -match '\{.*\}') {
                continue
            }
            
            # Track multi-line type
            if ($line -match '\{') {
                $inType = $true
                $currentType = $typeName
                $braceCount = 1
            } else {
                # Opening brace might be on next line
                $skipUntilBrace = $true
                $currentType = $typeName
            }
        }
    }
    elseif ($skipUntilBrace) {
        # Looking for opening brace of a type declaration
        if ($line -match '\{') {
            $inType = $true
            $skipUntilBrace = $false
            $braceCount = 1
        }
        $outputLines += $line
    }
    elseif ($inType) {
        # Inside a type definition
        if ($seenTypes[$currentType] -lt $i) {
            # This is part of a duplicate - skip it
            $openBraces = ([regex]::Matches($line, '\{').Count)
            $closeBraces = ([regex]::Matches($line, '\}').Count)
            $braceCount += $openBraces - $closeBraces
            
            if ($braceCount -eq 0) {
                $inType = $false
            }
            continue
        } else {
            # This is part of the first occurrence - keep it
            $openBraces = ([regex]::Matches($line, '\{').Count)
            $closeBraces = ([regex]::Matches($line, '\}').Count)
            $braceCount += $openBraces - $closeBraces
            
            if ($braceCount -eq 0) {
                $inType = $false
            }
            $outputLines += $line
        }
    }
    else {
        # Not in a type declaration, add the line
        $outputLines += $line
    }
}

# Write the cleaned content back
Set-Content -Path $typesFile -Value $outputLines

Write-Host "Cleaned types.go - removed $($seenTypes.Count) duplicate declarations" -ForegroundColor Green

# Test the build
Write-Host "`nTesting build..." -ForegroundColor Cyan
$errors = go build ./cmd/dashboard-api 2>&1 | Out-String

if ($errors -match "redeclared") {
    Write-Host "Still have redeclaration errors:" -ForegroundColor Red
    $errors -split "`n" | Where-Object { $_ -match "redeclared" } | ForEach-Object {
        Write-Host "  $_" -ForegroundColor Red
    }
} else {
    Write-Host "SUCCESS! Build completed without redeclaration errors!" -ForegroundColor Green
}