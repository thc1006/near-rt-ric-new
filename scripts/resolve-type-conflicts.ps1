# Type Conflict Resolution Orchestrator for Dashboard API (PowerShell Version)
# This script systematically resolves all Go type redeclaration errors

$ErrorActionPreference = "Stop"

Write-Host "===========================================" -ForegroundColor Cyan
Write-Host "Dashboard API Type Consolidation Orchestrator" -ForegroundColor Cyan
Write-Host "===========================================" -ForegroundColor Cyan

$DASHBOARD_DIR = "pkg\dashboard"
$TYPES_FILE = "$DASHBOARD_DIR\types.go"
$BACKUP_DIR = "backup_$(Get-Date -Format 'yyyyMMdd_HHmmss')"

function Write-Success {
    param($Message)
    Write-Host "[✓] " -ForegroundColor Green -NoNewline
    Write-Host $Message
}

function Write-Error {
    param($Message)
    Write-Host "[✗] " -ForegroundColor Red -NoNewline
    Write-Host $Message
}

function Write-Warning {
    param($Message)
    Write-Host "[!] " -ForegroundColor Yellow -NoNewline
    Write-Host $Message
}

# Phase 1: Backup
Write-Host "`nPhase 1: Creating backup..." -ForegroundColor Yellow
New-Item -ItemType Directory -Path $BACKUP_DIR -Force | Out-Null
Copy-Item -Path $DASHBOARD_DIR -Destination $BACKUP_DIR -Recurse -Force
Write-Success "Backup created in $BACKUP_DIR"

# Phase 2: Analysis
Write-Host "`nPhase 2: Analyzing type conflicts..." -ForegroundColor Yellow

# Find all type declarations
Write-Host "Scanning for duplicate type declarations..."
$typeLocations = @{}
$duplicateTypes = @{}

Get-ChildItem "$DASHBOARD_DIR\*.go" | ForEach-Object {
    $file = $_.Name
    $content = Get-Content $_.FullName
    
    foreach ($line in $content) {
        if ($line -match '^type\s+([A-Za-z][A-Za-z0-9]*)\s+(struct|interface)') {
            $typeName = $matches[1]
            if (-not $typeLocations.ContainsKey($typeName)) {
                $typeLocations[$typeName] = @($file)
            } else {
                $typeLocations[$typeName] += $file
                $duplicateTypes[$typeName] = $true
            }
        }
    }
}

# Report duplicate types
Write-Host "`nFound duplicate type declarations:"
foreach ($typeName in $duplicateTypes.Keys) {
    Write-Host "  - $typeName in: $($typeLocations[$typeName] -join ', ')"
}

# Phase 3: Get current build errors
Write-Host "`nPhase 3: Getting current build errors..." -ForegroundColor Yellow
$buildErrors = go build ./cmd/dashboard-api 2>&1 | Out-String
$redeclaredTypes = @()

# Parse error messages for redeclared types
$errorLines = $buildErrors -split "`n"
foreach ($line in $errorLines) {
    if ($line -match '(\w+)\s+redeclared in this block') {
        $redeclaredTypes += $matches[1]
    }
}

$redeclaredTypes = $redeclaredTypes | Select-Object -Unique
Write-Host "Types to consolidate: $($redeclaredTypes -join ', ')"

# Phase 4: Remove duplicates from source files
Write-Host "`nPhase 4: Removing duplicate declarations from source files..." -ForegroundColor Yellow

function Remove-TypeFromFile {
    param(
        [string]$FilePath,
        [string]$TypeName
    )
    
    # Skip types.go
    if ($FilePath -like "*types.go") {
        return
    }
    
    Write-Host "  Removing $TypeName from $(Split-Path -Leaf $FilePath)..."
    
    $content = Get-Content $FilePath
    $newContent = @()
    $inType = $false
    $braceCount = 0
    
    for ($i = 0; $i -lt $content.Count; $i++) {
        $line = $content[$i]
        
        if ($line -match "^type\s+$TypeName\s+(struct|interface)") {
            $inType = $true
            if ($line -match "{") { $braceCount++ }
            if ($line -match "}") { 
                $braceCount--
                if ($braceCount -eq 0) { $inType = $false }
            }
            continue
        }
        
        if ($inType) {
            if ($line -match "{") { $braceCount++ }
            if ($line -match "}") { 
                $braceCount--
                if ($braceCount -eq 0) { $inType = $false }
            }
            continue
        }
        
        $newContent += $line
    }
    
    Set-Content -Path $FilePath -Value $newContent
}

# Remove each redeclared type from non-types.go files
foreach ($typeName in $redeclaredTypes) {
    # Skip WorkType constants
    if ($typeName -like "WorkType*") {
        continue
    }
    
    $files = Get-ChildItem "$DASHBOARD_DIR\*.go" | Where-Object {
        $_.Name -ne "types.go" -and 
        (Get-Content $_.FullName | Select-String -Pattern "type\s+$typeName\s+(struct|interface)" -Quiet)
    }
    
    foreach ($file in $files) {
        Remove-TypeFromFile -FilePath $file.FullName -TypeName $typeName
    }
}

# Phase 5: Handle WorkType constants
Write-Host "`nPhase 5: Handling WorkType constants..." -ForegroundColor Yellow

$workTypeConstants = @(
    "WorkTypeE2APMessage",
    "WorkTypeSubscription", 
    "WorkTypeIndication",
    "WorkTypeControl",
    "WorkTypePolicyUpdate"
)

foreach ($constName in $workTypeConstants) {
    $files = Get-ChildItem "$DASHBOARD_DIR\*.go" | Where-Object {
        $_.Name -ne "types.go" -and
        (Get-Content $_.FullName | Select-String -Pattern $constName -Quiet)
    }
    
    foreach ($file in $files) {
        Write-Host "  Removing $constName from $(Split-Path -Leaf $file.FullName)..."
        $content = Get-Content $file.FullName
        $newContent = $content | Where-Object { $_ -notmatch "$constName\s*WorkType" }
        Set-Content -Path $file.FullName -Value $newContent
    }
}

# Phase 6: Clean up types.go to remove internal duplicates
Write-Host "`nPhase 6: Cleaning up types.go..." -ForegroundColor Yellow

$typesContent = Get-Content $TYPES_FILE
$seenTypes = @{}
$seenConsts = @{}
$cleanContent = @()
$inType = $false
$inConstBlock = $false
$braceCount = 0

for ($i = 0; $i -lt $typesContent.Count; $i++) {
    $line = $typesContent[$i]
    
    # Handle type declarations
    if ($line -match '^type\s+([A-Za-z][A-Za-z0-9]*)\s+(struct|interface)') {
        $typeName = $matches[1]
        
        if ($seenTypes.ContainsKey($typeName)) {
            # Skip duplicate type
            $inType = $true
            if ($line -match "{") { $braceCount++ }
            if ($line -match "}") { 
                $braceCount--
                if ($braceCount -eq 0) { $inType = $false }
            }
            continue
        }
        
        $seenTypes[$typeName] = $true
        $cleanContent += $line
        
        if ($line -match "{" -and $line -match "}") {
            # Single line type
            continue
        } elseif ($line -match "{") {
            $inType = $true
            $braceCount = 1
        }
        continue
    }
    
    # Handle const blocks
    if ($line -match '^const\s*\(') {
        $inConstBlock = $true
        $cleanContent += $line
        continue
    }
    
    if ($inConstBlock -and $line -match '^\)') {
        $inConstBlock = $false
        $cleanContent += $line
        continue
    }
    
    # Handle constants in const block
    if ($inConstBlock -and $line -match '^\s*([A-Za-z][A-Za-z0-9]*)\s*WorkType') {
        $constName = $matches[1]
        if (-not $seenConsts.ContainsKey($constName)) {
            $seenConsts[$constName] = $true
            $cleanContent += $line
        }
        continue
    }
    
    # Handle multi-line type body
    if ($inType) {
        $cleanContent += $line
        if ($line -match "{") { $braceCount++ }
        if ($line -match "}") { 
            $braceCount--
            if ($braceCount -eq 0) { $inType = $false }
        }
        continue
    }
    
    # Add all other lines
    $cleanContent += $line
}

Set-Content -Path $TYPES_FILE -Value $cleanContent
Write-Success "Cleaned up types.go"

# Phase 7: Test build
Write-Host "`nPhase 7: Testing build..." -ForegroundColor Yellow

$buildOutput = go build ./cmd/dashboard-api 2>&1 | Out-String
if ($buildOutput -match "redeclared") {
    Write-Error "Build still has redeclaration errors. Analyzing..."
    
    Write-Host "`nRemaining errors:" -ForegroundColor Red
    $buildOutput -split "`n" | Where-Object { $_ -match "redeclared|previous declaration" } | ForEach-Object {
        Write-Host "  $_" -ForegroundColor Red
    }
    
    # Try iterative fix
    Write-Host "`nAttempting iterative fix..." -ForegroundColor Yellow
    
    $remainingErrors = $buildOutput -split "`n" | Where-Object { $_ -match "(\w+)\s+redeclared in this block" }
    foreach ($error in $remainingErrors) {
        if ($error -match '(\w+)\s+redeclared') {
            $typeName = $matches[1]
            Write-Warning "Fixing remaining duplicate: $typeName"
            
            # Remove from all files except types.go
            Get-ChildItem "$DASHBOARD_DIR\*.go" | Where-Object { $_.Name -ne "types.go" } | ForEach-Object {
                Remove-TypeFromFile -FilePath $_.FullName -TypeName $typeName
            }
        }
    }
    
    # Test again
    $buildOutput = go build ./cmd/dashboard-api 2>&1 | Out-String
    if ($buildOutput -match "redeclared") {
        Write-Error "Some errors remain. Manual intervention needed."
    } else {
        Write-Success "Build successful after iterative fixes!"
    }
} else {
    Write-Success "BUILD SUCCESSFUL! All type conflicts resolved."
}

# Phase 8: Final validation
Write-Host "`nPhase 8: Final validation..." -ForegroundColor Yellow

$totalFiles = (Get-ChildItem "$DASHBOARD_DIR\*.go").Count
$typesInTypesGo = (Get-Content $TYPES_FILE | Select-String -Pattern '^type\s+' -AllMatches).Matches.Count

Write-Host "Type declaration summary:"
Write-Host "  Total Go files: $totalFiles"
Write-Host "  Types in types.go: $typesInTypesGo"

# Phase 9: Generate report
Write-Host "`nPhase 9: Generating report..." -ForegroundColor Yellow

$reportFile = "type_consolidation_report_$(Get-Date -Format 'yyyyMMdd_HHmmss').txt"
$report = @"
Type Consolidation Report
=========================
Date: $(Get-Date)

Files Modified:
$(Get-ChildItem "$DASHBOARD_DIR\*.go" | ForEach-Object {
    $original = "$BACKUP_DIR\dashboard\$($_.Name)"
    if (Test-Path $original) {
        $diff = Compare-Object (Get-Content $_.FullName) (Get-Content $original) -PassThru
        if ($diff) {
            "  - $($_.Name)"
        }
    }
} | Out-String)

Types Consolidated:
$($redeclaredTypes | ForEach-Object { "  - $_" } | Out-String)

Build Status: $(if ($buildOutput -match "redeclared") { "FAILED" } else { "SUCCESS" })
"@

Set-Content -Path $reportFile -Value $report
Write-Success "Report generated: $reportFile"

Write-Host "`n===========================================" -ForegroundColor Cyan
Write-Host "Orchestration complete!" -ForegroundColor Green
Write-Host "Backup location: $BACKUP_DIR" -ForegroundColor Cyan
Write-Host "Report: $reportFile" -ForegroundColor Cyan
Write-Host "===========================================" -ForegroundColor Cyan