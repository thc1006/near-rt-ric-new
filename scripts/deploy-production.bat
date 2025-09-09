@echo off
REM Production deployment script for O-RAN Near-RT RIC (Windows)

setlocal enabledelayedexpansion

REM Configuration
set COMPOSE_FILE=docker-compose.production.yml
set PROJECT_NAME=oran-near-rt-ric
if "%VERSION%"=="" set VERSION=v1.0.0
if "%ENV_FILE%"=="" set ENV_FILE=.env.production

echo.
echo ========================================
echo O-RAN Near-RT RIC Production Deployment
echo ========================================
echo.

REM Check if Docker is running
docker info >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Docker is not running. Please start Docker Desktop and try again.
    pause
    exit /b 1
)

REM Check if Docker Compose is available
docker compose version >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Docker Compose is not available. Please install Docker Desktop with Compose.
    pause
    exit /b 1
)

REM Check if compose file exists
if not exist "%COMPOSE_FILE%" (
    echo [ERROR] Compose file %COMPOSE_FILE% not found!
    pause
    exit /b 1
)

echo [INFO] Checking if environment file exists...
if not exist "%ENV_FILE%" (
    echo [INFO] Creating environment file: %ENV_FILE%
    (
        echo # O-RAN Near-RT RIC Production Environment
        echo.
        echo # Application Version
        echo VERSION=%VERSION%
        echo.
        echo # Database Configuration
        echo DB_PASSWORD=secure_password_123
        echo POSTGRES_PASSWORD=secure_password_123
        echo.
        echo # Redis Configuration
        echo REDIS_PASSWORD=secure_redis_123
        echo.
        echo # Security
        echo JWT_SECRET=your_jwt_secret_key_here_replace_in_production
        echo GRAFANA_SECRET=grafana_secret_key
        echo GRAFANA_PASSWORD=admin123
        echo.
        echo # Monitoring
        echo JAEGER_AGENT_HOST=jaeger
        echo JAEGER_AGENT_PORT=6831
        echo.
        echo # Environment
        echo NODE_ENV=production
        echo GO_ENV=production
        echo LOG_LEVEL=info
    ) > "%ENV_FILE%"
    echo [SUCCESS] Environment file created: %ENV_FILE%
    echo [WARNING] Please review and update %ENV_FILE% with secure passwords!
    echo.
)

REM Create certificates directory
if not exist "certs" mkdir certs

echo [INFO] Building Docker images...
docker compose -f "%COMPOSE_FILE%" --env-file "%ENV_FILE%" -p "%PROJECT_NAME%" build --parallel
if errorlevel 1 (
    echo [ERROR] Failed to build Docker images
    pause
    exit /b 1
)

echo [INFO] Starting infrastructure services...
docker compose -f "%COMPOSE_FILE%" --env-file "%ENV_FILE%" -p "%PROJECT_NAME%" up -d postgres redis elasticsearch
if errorlevel 1 (
    echo [ERROR] Failed to start infrastructure services
    pause
    exit /b 1
)

echo [INFO] Waiting for databases to be ready...
timeout /t 30 /nobreak >nul

echo [INFO] Starting monitoring services...
docker compose -f "%COMPOSE_FILE%" --env-file "%ENV_FILE%" -p "%PROJECT_NAME%" up -d prometheus grafana jaeger

echo [INFO] Starting application services...
docker compose -f "%COMPOSE_FILE%" --env-file "%ENV_FILE%" -p "%PROJECT_NAME%" up -d dashboard-api ric-core xapp-manager

echo [INFO] Starting web services...
docker compose -f "%COMPOSE_FILE%" --env-file "%ENV_FILE%" -p "%PROJECT_NAME%" up -d web-dashboard nginx

echo [INFO] Waiting for services to become healthy...
set /a attempts=0
:wait_loop
if %attempts% geq 30 (
    echo [WARNING] Services may still be starting up. Check manually.
    goto show_status
)

curl -f http://localhost:80/health >nul 2>&1
if errorlevel 1 (
    set /a attempts+=1
    echo Attempt !attempts!/30 - waiting for services...
    timeout /t 10 /nobreak >nul
    goto wait_loop
)

:show_status
echo.
echo ========================================
echo Deployment Status
echo ========================================
docker compose -f "%COMPOSE_FILE%" --env-file "%ENV_FILE%" -p "%PROJECT_NAME%" ps

echo.
echo ========================================
echo Service URLs
echo ========================================
echo 🌐 O-RAN Dashboard:    http://localhost:3000
echo 📊 Grafana:           http://localhost:3001
echo 🔍 Prometheus:        http://localhost:9090
echo 🕵️  Jaeger:            http://localhost:16686
echo 🔌 API Endpoint:      http://localhost:8080
echo 🏥 Health Check:      http://localhost:80/health
echo.
echo ========================================
echo Default Credentials
echo ========================================
echo Grafana: admin / admin123
echo.

echo [SUCCESS] 🎉 O-RAN Near-RT RIC deployed successfully!
echo.
echo Opening dashboard in your default browser...
start http://localhost:3000

echo.
echo Press any key to view logs, or Ctrl+C to exit...
pause >nul

REM Show logs
docker compose -f "%COMPOSE_FILE%" --env-file "%ENV_FILE%" -p "%PROJECT_NAME%" logs -f

endlocal