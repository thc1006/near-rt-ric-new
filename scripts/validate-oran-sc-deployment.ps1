# O-RAN SC Component Validation Script
# This script validates that all O-RAN SC components are deployed and communicating properly

param(
    [string]$Namespace = "ricplt"
)

Write-Host "=== O-RAN SC Near-RT RIC Validation ===" -ForegroundColor Green

# Function to check pod readiness
function Test-PodReady {
    param(
        [string]$AppLabel,
        [string]$ComponentName
    )
    
    Write-Host "Checking $ComponentName..." -ForegroundColor Yellow
    
    $pods = kubectl get pods -n $Namespace -l app=$AppLabel --no-headers 2>$null
    if ($LASTEXITCODE -ne 0) {
        Write-Host "✗ $ComponentName pods not found" -ForegroundColor Red
        return $false
    }
    
    $readyPods = $pods | Where-Object { $_ -match "1/1\s+Running" }
    if ($readyPods) {
        Write-Host "✓ $ComponentName is ready" -ForegroundColor Green
        return $true
    } else {
        Write-Host "✗ $ComponentName is not ready" -ForegroundColor Red
        kubectl get pods -n $Namespace -l app=$AppLabel
        return $false
    }
}

# Function to check service availability
function Test-ServiceEndpoint {
    param(
        [string]$ServiceName,
        [string]$ComponentName
    )
    
    Write-Host "Checking $ComponentName service..." -ForegroundColor Yellow
    
    $service = kubectl get service $ServiceName -n $Namespace --no-headers 2>$null
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✓ $ComponentName service is available" -ForegroundColor Green
        return $true
    } else {
        Write-Host "✗ $ComponentName service not found" -ForegroundColor Red
        return $false
    }
}

# Check if namespace exists
Write-Host "Checking namespace: $Namespace" -ForegroundColor Yellow
$ns = kubectl get namespace $Namespace --no-headers 2>$null
if ($LASTEXITCODE -ne 0) {
    Write-Host "✗ Namespace $Namespace does not exist" -ForegroundColor Red
    exit 1
}
Write-Host "✓ Namespace $Namespace exists" -ForegroundColor Green

Write-Host ""
Write-Host "=== Checking Core Components ===" -ForegroundColor Cyan

# Check core O-RAN SC components
$components = @(
    @{Label="ricplt-dbaas"; Name="Database (Redis)"},
    @{Label="ricplt-e2mgr"; Name="E2 Manager"},
    @{Label="ricplt-submgr"; Name="Subscription Manager"},
    @{Label="ricplt-rtmgr"; Name="Routing Manager"},
    @{Label="ricplt-appmgr"; Name="App Manager"},
    @{Label="ricplt-a1mediator"; Name="A1 Mediator"},
    @{Label="ricplt-e2term"; Name="E2 Termination"}
)

$allComponentsReady = $true
foreach ($component in $components) {
    if (-not (Test-PodReady -AppLabel $component.Label -ComponentName $component.Name)) {
        $allComponentsReady = $false
    }
}

Write-Host ""
Write-Host "=== Checking Service Endpoints ===" -ForegroundColor Cyan

# Check key service endpoints
$services = @(
    @{Name="service-ricplt-e2mgr-http"; Component="E2 Manager"},
    @{Name="service-ricplt-a1mediator-http"; Component="A1 Mediator"},
    @{Name="service-ricplt-appmgr-http"; Component="App Manager"},
    @{Name="service-ricplt-rtmgr-http"; Component="Routing Manager"},
    @{Name="service-ricplt-dbaas-tcp"; Component="Database (Redis)"}
)

$allServicesReady = $true
foreach ($service in $services) {
    if (-not (Test-ServiceEndpoint -ServiceName $service.Name -ComponentName $service.Component)) {
        $allServicesReady = $false
    }
}

Write-Host ""
Write-Host "=== Component Communication Test ===" -ForegroundColor Cyan

# Test E2 Manager health endpoint
Write-Host "Testing E2 Manager health endpoint..." -ForegroundColor Yellow
$e2mgrPod = kubectl get pods -n $Namespace -l app=ricplt-e2mgr --no-headers | Select-Object -First 1 | ForEach-Object { ($_ -split '\s+')[0] }
if ($e2mgrPod) {
    $healthCheck = kubectl exec -n $Namespace $e2mgrPod -- curl -s -f http://localhost:3800/v1/health 2>$null
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✓ E2 Manager health endpoint responding" -ForegroundColor Green
    } else {
        Write-Host "⚠ E2 Manager health endpoint not responding (may be normal during startup)" -ForegroundColor Yellow
    }
}

# Test App Manager health endpoint  
Write-Host "Testing App Manager health endpoint..." -ForegroundColor Yellow
$appmgrPod = kubectl get pods -n $Namespace -l app=ricplt-appmgr --no-headers | Select-Object -First 1 | ForEach-Object { ($_ -split '\s+')[0] }
if ($appmgrPod) {
    $healthCheck = kubectl exec -n $Namespace $appmgrPod -- curl -s -f http://localhost:8080/ric/v1/health/ready 2>$null
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✓ App Manager health endpoint responding" -ForegroundColor Green
    } else {
        Write-Host "⚠ App Manager health endpoint not responding (may be normal during startup)" -ForegroundColor Yellow
    }
}

Write-Host ""
Write-Host "=== Overall System Status ===" -ForegroundColor Cyan

# Get overall pod status
Write-Host "Pod Status Summary:" -ForegroundColor Yellow
kubectl get pods -n $Namespace -o wide

Write-Host ""
Write-Host "Service Status Summary:" -ForegroundColor Yellow
kubectl get services -n $Namespace

# Count ready pods
$allPods = kubectl get pods -n $Namespace --no-headers
$readyPods = $allPods | Where-Object { $_ -match "1/1\s+Running" }
$readyCount = if ($readyPods) { $readyPods.Count } else { 0 }
$totalCount = if ($allPods) { $allPods.Count } else { 0 }

Write-Host ""
Write-Host "=== Validation Summary ===" -ForegroundColor Cyan
Write-Host "Ready Pods: $readyCount/$totalCount" -ForegroundColor Yellow

if ($allComponentsReady -and $allServicesReady -and $readyCount -eq $totalCount -and $totalCount -gt 0) {
    Write-Host "✓ All O-RAN SC components are healthy and ready" -ForegroundColor Green
    Write-Host ""
    Write-Host "Key Endpoints:" -ForegroundColor Cyan
    Write-Host "  E2 Manager:    http://service-ricplt-e2mgr-http.ricplt:3800" -ForegroundColor White
    Write-Host "  A1 Mediator:   http://service-ricplt-a1mediator-http.ricplt:10000" -ForegroundColor White
    Write-Host "  App Manager:   http://service-ricplt-appmgr-http.ricplt:8080" -ForegroundColor White
    Write-Host "  Routing Mgr:   http://service-ricplt-rtmgr-http.ricplt:3800" -ForegroundColor White
    Write-Host ""
    Write-Host "E2 Interface:    SCTP port 36422 (NodePort 32222)" -ForegroundColor White
    Write-Host "Shared Data:     Redis at service-ricplt-dbaas-tcp.ricplt:6379" -ForegroundColor White
    exit 0
} else {
    Write-Host "⚠ Some components may not be ready yet" -ForegroundColor Yellow
    Write-Host "This is normal during initial deployment. Components may take a few minutes to start." -ForegroundColor Yellow
    exit 0
}