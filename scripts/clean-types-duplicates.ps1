# Script to remove duplicate type declarations within types.go

$typesFile = "pkg\dashboard\types.go"
$content = Get-Content $typesFile

# Track seen types
$seenTypes = @{}
$outputLines = @()
$inType = $false
$typeStartLine = -1
$currentType = ""
$braceCount = 0

for ($i = 0; $i -lt $content.Count; $i++) {
    $line = $content[$i]
    
    # Check for type declaration
    if ($line -match '^type\s+([A-Za-z][A-Za-z0-9]*)\s+(struct|interface)') {
        $typeName = $matches[1]
        
        if ($seenTypes.ContainsKey($typeName)) {
            Write-Host "Skipping duplicate: $typeName at line $($i + 1)"
            # Skip this type declaration
            $inType = $true
            $currentType = $typeName
            $typeStartLine = $i
            
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
            $seenTypes[$typeName] = $true
            $outputLines += $line
            
            # Check if it's a single-line declaration
            if ($line -match '\{.*\}') {
                continue
            }
            
            # Track multi-line type
            if ($line -match '\{') {
                $inType = $true
                $currentType = $typeName
                $braceCount = 1
            }
        }
    }
    elseif ($inType) {
        # Count braces
        $openBraces = ([regex]::Matches($line, '\{').Count)
        $closeBraces = ([regex]::Matches($line, '\}').Count)
        $braceCount += $openBraces - $closeBraces
        
        # Check if type definition is complete
        if ($braceCount -eq 0) {
            $inType = $false
        }
        
        # Skip lines that are part of duplicate type
        if ($seenTypes.ContainsKey($currentType) -and $typeStartLine -ge 0) {
            continue
        }
        
        $outputLines += $line
    }
    else {
        # Not in a type declaration, add the line
        $outputLines += $line
    }
}

# Write cleaned content back
Set-Content -Path $typesFile -Value $outputLines

Write-Host "Cleaned types.go - removed duplicate declarations"
Write-Host "Total types seen: $($seenTypes.Count)"