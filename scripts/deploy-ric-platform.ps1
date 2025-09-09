# O-RAN Near-RT RIC Complete Deployment Script for Windows
# This script orchestrates the full deployment of the RIC platform

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = Split-Path -Parent $ScriptDir
$DeploymentDir = Join-Path $ProjectRoot "deployments"

Write-Host "==========================================" -ForegroundColor Green
Write-Host "O-RAN Near-RT RIC Deployment Orchestrator" -ForegroundColor Green
Write-Host "==========================================" -ForegroundColor Green

function Write-Status {
    param($Message)
    Write-Host "[$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')] " -NoNewline -ForegroundColor Yellow
    Write-Host $Message
}

function Check-Prerequisites {
    Write-Status "Checking prerequisites..."
    
    # Check kubectl
    try {
        kubectl version --client | Out-Null
    } catch {
        Write-Host "kubectl not found! Please install kubectl." -ForegroundColor Red
        exit 1
    }
    
    # Check cluster connection
    try {
        kubectl cluster-info | Out-Null
    } catch {
        Write-Host "Cannot connect to Kubernetes cluster!" -ForegroundColor Red
        exit 1
    }
    
    Write-Status "Prerequisites check passed"
}

function Install-IngressController {
    Write-Status "Checking NGINX Ingress Controller..."
    
    $ingressExists = kubectl get namespace ingress-nginx 2>$null
    if ($ingressExists) {
        Write-Status "NGINX Ingress Controller already installed"
    } else {
        Write-Status "Installing NGINX Ingress Controller..."
        kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.8.2/deploy/static/provider/cloud/deploy.yaml
        Write-Status "Waiting for Ingress Controller to be ready..."
        Start-Sleep -Seconds 10
    }
}

function Cleanup-Existing {
    Write-Status "Cleaning up existing deployments..."
    
    $namespaces = @("ricplt", "ricxapp", "monitoring")
    foreach ($ns in $namespaces) {
        $nsExists = kubectl get namespace $ns 2>$null
        if ($nsExists) {
            Write-Status "Deleting namespace $ns..."
            kubectl delete namespace $ns --wait=false 2>$null
        }
    }
    
    Write-Status "Waiting for cleanup to complete..."
    Start-Sleep -Seconds 20
}

function Deploy-RICPlatform {
    Write-Status "Deploying O-RAN Near-RT RIC Platform..."
    
    $deploymentFile = Join-Path $DeploymentDir "complete-ric-deployment.yaml"
    kubectl apply -f $deploymentFile
    
    Write-Status "Waiting for deployments to be ready..."
    Start-Sleep -Seconds 10
    
    # Wait for pods to be ready
    Write-Status "Waiting for pods to start..."
    $maxRetries = 30
    $retryCount = 0
    
    while ($retryCount -lt $maxRetries) {
        $pods = kubectl get pods -n ricplt --no-headers 2>$null
        if ($pods) {
            $runningPods = ($pods | Select-String "Running").Count
            $totalPods = ($pods | Measure-Object -Line).Lines
            Write-Status "Pods running: $runningPods/$totalPods"
            
            if ($runningPods -ge 5) {
                break
            }
        }
        Start-Sleep -Seconds 5
        $retryCount++
    }
}

function Verify-Deployment {
    Write-Status "Verifying deployment..."
    
    Write-Host "`n=== Namespace Status ===" -ForegroundColor Green
    kubectl get namespaces | Select-String -Pattern "ricplt|ricxapp|monitoring"
    
    Write-Host "`n=== RIC Platform Components ===" -ForegroundColor Green
    kubectl get pods -n ricplt
    
    Write-Host "`n=== xApps ===" -ForegroundColor Green
    kubectl get pods -n ricxapp
    
    Write-Host "`n=== Services ===" -ForegroundColor Green
    kubectl get services -n ricplt
    
    Write-Host "`n=== Ingress ===" -ForegroundColor Green
    kubectl get ingress -n ricplt 2>$null
}

function Get-AccessInfo {
    Write-Status "Getting access information..."
    
    # Get NodePort
    $nodePort = kubectl get service ric-dashboard-nodeport -n ricplt -o jsonpath='{.spec.ports[0].nodePort}'
    
    # Get Node IP
    $nodeIP = "localhost"
    
    Write-Host "`n==========================================" -ForegroundColor Green
    Write-Host "Deployment Complete!" -ForegroundColor Green
    Write-Host "==========================================" -ForegroundColor Green
    Write-Host "`nAccess the RIC Dashboard:" -ForegroundColor Yellow
    Write-Host "  Browser URL: " -NoNewline
    Write-Host "http://${nodeIP}:${nodePort}" -ForegroundColor Green
    Write-Host "`nAPI Endpoints:" -ForegroundColor Yellow
    Write-Host "  Status API: " -NoNewline
    Write-Host "http://${nodeIP}:${nodePort}/api/status" -ForegroundColor Green
    Write-Host "  Health Check: " -NoNewline
    Write-Host "http://${nodeIP}:${nodePort}/health" -ForegroundColor Green
    Write-Host "`nPort Forwarding (if needed):" -ForegroundColor Yellow
    Write-Host "  kubectl port-forward -n ricplt svc/ric-dashboard-api 8080:8080"
    
    return $nodePort
}

function Setup-PortForwarding {
    Write-Status "Setting up port forwarding for easy access..."
    
    # Start port forwarding in a new window
    $portForwardCmd = "kubectl port-forward -n ricplt svc/ric-dashboard-api 8080:8080"
    Start-Process -FilePath "cmd.exe" -ArgumentList "/c", $portForwardCmd -PassThru
    
    Write-Host "`nPort forwarding established!" -ForegroundColor Green
    Write-Host "Dashboard accessible at: " -NoNewline
    Write-Host "http://localhost:8080" -ForegroundColor Green
}

function Open-Dashboard {
    param($Port)
    
    $url = "http://localhost:$Port"
    Write-Status "Opening dashboard in default browser..."
    Start-Process $url
}

# Main deployment flow
function Main {
    Write-Host "Starting O-RAN Near-RT RIC deployment...`n" -ForegroundColor Yellow
    
    # Check prerequisites
    Check-Prerequisites
    
    # Ask user for clean deployment
    $cleanup = Read-Host "Do you want to clean up existing deployments? (y/n)"
    if ($cleanup -eq 'y') {
        Cleanup-Existing
    }
    
    # Install ingress controller
    Install-IngressController
    
    # Deploy RIC platform
    Deploy-RICPlatform
    
    # Verify deployment
    Verify-Deployment
    
    # Get access information
    $nodePort = Get-AccessInfo
    
    # Ask if user wants port forwarding
    $portForward = Read-Host "`nDo you want to setup port forwarding for easy access? (y/n)"
    if ($portForward -eq 'y') {
        Setup-PortForwarding
        Start-Sleep -Seconds 3
    }
    
    # Ask if user wants to open dashboard
    $openBrowser = Read-Host "`nDo you want to open the dashboard in your browser? (y/n)"
    if ($openBrowser -eq 'y') {
        Open-Dashboard -Port $nodePort
    }
    
    Write-Host "`nDeployment orchestration complete!" -ForegroundColor Green
    Write-Host "Press any key to exit..."
    $null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
}

# Run main function
Main