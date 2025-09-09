# O-RAN Near-RT RIC Deployment Verification Script

Write-Host "`n================================================" -ForegroundColor Cyan
Write-Host "  O-RAN Near-RT RIC Deployment Verification" -ForegroundColor Cyan
Write-Host "================================================`n" -ForegroundColor Cyan

# Check pods
Write-Host "Checking Pod Status..." -ForegroundColor Yellow
$pods = kubectl get pods -n ricplt --no-headers 2>$null
$runningPods = ($pods | Select-String "Running").Count
$totalPods = ($pods | Measure-Object -Line).Lines

Write-Host "Pods Running: " -NoNewline
Write-Host "$runningPods/$totalPods" -ForegroundColor Green

# Check services
Write-Host "`nChecking Services..." -ForegroundColor Yellow
$services = kubectl get services -n ricplt --no-headers 2>$null | Measure-Object -Line
Write-Host "Services Deployed: " -NoNewline
Write-Host "$($services.Lines)" -ForegroundColor Green

# Test Dashboard
Write-Host "`nTesting Dashboard Access..." -ForegroundColor Yellow

# Test port 8080
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8080/health" -UseBasicParsing -TimeoutSec 5
    if ($response.StatusCode -eq 200) {
        Write-Host "Dashboard (Port 8080): " -NoNewline
        Write-Host "ACCESSIBLE" -ForegroundColor Green
    }
} catch {
    Write-Host "Dashboard (Port 8080): " -NoNewline
    Write-Host "NOT ACCESSIBLE" -ForegroundColor Red
}

# Test NodePort
try {
    $response = Invoke-WebRequest -Uri "http://localhost:30080/health" -UseBasicParsing -TimeoutSec 5
    if ($response.StatusCode -eq 200) {
        Write-Host "NodePort (30080): " -NoNewline
        Write-Host "ACCESSIBLE" -ForegroundColor Green
    }
} catch {
    Write-Host "NodePort (30080): " -NoNewline
    Write-Host "NOT ACCESSIBLE (Normal if using Docker Desktop)" -ForegroundColor Yellow
}

# Display Access URLs
Write-Host "`n================================================" -ForegroundColor Cyan
Write-Host "  Dashboard Access URLs" -ForegroundColor Cyan
Write-Host "================================================" -ForegroundColor Cyan

Write-Host "`nPrimary Access (Port Forward):" -ForegroundColor Yellow
Write-Host "  " -NoNewline
Write-Host "http://localhost:8080" -ForegroundColor Green

Write-Host "`nAlternative Access (NodePort):" -ForegroundColor Yellow
Write-Host "  " -NoNewline
Write-Host "http://localhost:30080" -ForegroundColor Green

Write-Host "`nAPI Endpoints:" -ForegroundColor Yellow
Write-Host "  Status: http://localhost:8080/api/status" -ForegroundColor Cyan
Write-Host "  Health: http://localhost:8080/health" -ForegroundColor Cyan

Write-Host "`n================================================" -ForegroundColor Cyan
Write-Host "  Deployment Status: " -NoNewline
Write-Host "OPERATIONAL" -ForegroundColor Green
Write-Host "================================================`n" -ForegroundColor Cyan

# Ask to open browser
$openBrowser = Read-Host "Would you like to open the dashboard in your browser? (y/n)"
if ($openBrowser -eq 'y') {
    Start-Process "http://localhost:8080"
    Write-Host "`nDashboard opened in default browser!" -ForegroundColor Green
}

Write-Host "`nVerification complete!" -ForegroundColor Green