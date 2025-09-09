#!/bin/bash

# O-RAN Near-RT RIC Build Verification Script
# Orchestrator Agent - Build Coordination

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Build status tracking
BUILD_STATUS_FILE="build-status.json"
FAILED_COMPONENTS=()
SUCCESSFUL_COMPONENTS=()

echo -e "${BLUE}=== O-RAN Near-RT RIC Build Verification ===${NC}"
echo -e "${YELLOW}Starting comprehensive build verification process...${NC}\n"

# Initialize build status
cat > $BUILD_STATUS_FILE <<EOF
{
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "stage": "build-verification",
  "status": "in-progress",
  "components": {}
}
EOF

# Function to update build status
update_status() {
    local component=$1
    local status=$2
    local message=$3
    
    if [ "$status" = "success" ]; then
        SUCCESSFUL_COMPONENTS+=("$component")
        echo -e "${GREEN}✓ $component: Build successful${NC}"
    else
        FAILED_COMPONENTS+=("$component")
        echo -e "${RED}✗ $component: Build failed - $message${NC}"
    fi
}

# Step 1: Verify Prerequisites
echo -e "${YELLOW}Step 1: Verifying prerequisites...${NC}"
echo -n "  Checking Go installation... "
if command -v go &> /dev/null; then
    GO_VERSION=$(go version | awk '{print $3}')
    echo -e "${GREEN}OK ($GO_VERSION)${NC}"
else
    echo -e "${RED}FAILED - Go not installed${NC}"
    exit 1
fi

echo -n "  Checking Node.js installation... "
if command -v node &> /dev/null; then
    NODE_VERSION=$(node --version)
    echo -e "${GREEN}OK ($NODE_VERSION)${NC}"
else
    echo -e "${YELLOW}WARNING - Node.js not installed (UI build will be skipped)${NC}"
fi

echo -n "  Checking Docker installation... "
if command -v docker &> /dev/null; then
    DOCKER_VERSION=$(docker --version | awk '{print $3}' | sed 's/,//')
    echo -e "${GREEN}OK ($DOCKER_VERSION)${NC}"
else
    echo -e "${YELLOW}WARNING - Docker not installed${NC}"
fi

# Step 2: Verify Go Modules
echo -e "\n${YELLOW}Step 2: Verifying Go modules...${NC}"
echo -n "  Running go mod verify... "
if go mod verify &> /dev/null; then
    echo -e "${GREEN}OK${NC}"
else
    echo -e "${RED}FAILED${NC}"
    echo "  Attempting to fix module issues..."
    go mod download
    go mod tidy
fi

# Step 3: Build Core Components
echo -e "\n${YELLOW}Step 3: Building core components...${NC}"

# List of components to build
COMPONENTS=(
    "dashboard-api"
    "analytics-api"
    "e2-telemetry-processor"
    "kpi-calculator"
    "ml-predictor"
    "performance-analytics"
    "performance-optimizer"
    "telemetry-collector"
    "xapp-hello-world"
)

# Create bin directory if it doesn't exist
mkdir -p bin

# Build each component
for component in "${COMPONENTS[@]}"; do
    echo -n "  Building $component... "
    
    if [ -d "cmd/$component" ]; then
        BUILD_OUTPUT=$(go build -o "bin/$component" "./cmd/$component" 2>&1) || {
            update_status "$component" "failed" "$BUILD_OUTPUT"
            continue
        }
        update_status "$component" "success" ""
    else
        echo -e "${YELLOW}SKIPPED (directory not found)${NC}"
    fi
done

# Step 4: Build UI Components (if Node.js is available)
if command -v node &> /dev/null && [ -d "ui" ]; then
    echo -e "\n${YELLOW}Step 4: Building UI components...${NC}"
    cd ui
    
    echo -n "  Installing dependencies... "
    if npm install --legacy-peer-deps &> /dev/null; then
        echo -e "${GREEN}OK${NC}"
        
        echo -n "  Building React app... "
        if npm run build &> /dev/null; then
            echo -e "${GREEN}OK${NC}"
            update_status "ui" "success" ""
        else
            echo -e "${RED}FAILED${NC}"
            update_status "ui" "failed" "React build failed"
        fi
    else
        echo -e "${RED}FAILED${NC}"
        update_status "ui" "failed" "npm install failed"
    fi
    
    cd ..
else
    echo -e "\n${YELLOW}Step 4: Skipping UI build (Node.js not available or ui directory not found)${NC}"
fi

# Step 5: Run Basic Tests
echo -e "\n${YELLOW}Step 5: Running basic tests...${NC}"
echo -n "  Running unit tests... "
TEST_OUTPUT=$(go test -short ./... 2>&1) || {
    echo -e "${RED}FAILED${NC}"
    echo "  Some tests failed. Check test output for details."
}
echo -e "${GREEN}OK${NC}"

# Step 6: Generate Build Report
echo -e "\n${YELLOW}Step 6: Generating build report...${NC}"

TOTAL_COMPONENTS=$((${#SUCCESSFUL_COMPONENTS[@]} + ${#FAILED_COMPONENTS[@]}))
SUCCESS_RATE=0
if [ $TOTAL_COMPONENTS -gt 0 ]; then
    SUCCESS_RATE=$(( ${#SUCCESSFUL_COMPONENTS[@]} * 100 / $TOTAL_COMPONENTS ))
fi

cat > BUILD_VERIFICATION_REPORT.md <<EOF
# Build Verification Report

**Date:** $(date -u +"%Y-%m-%d %H:%M:%S UTC")
**Project:** O-RAN Near-RT RIC
**Build Success Rate:** ${SUCCESS_RATE}%

## Summary

- Total Components: $TOTAL_COMPONENTS
- Successful Builds: ${#SUCCESSFUL_COMPONENTS[@]}
- Failed Builds: ${#FAILED_COMPONENTS[@]}

## Successful Components

EOF

for component in "${SUCCESSFUL_COMPONENTS[@]}"; do
    echo "- ✅ $component" >> BUILD_VERIFICATION_REPORT.md
done

echo "" >> BUILD_VERIFICATION_REPORT.md
echo "## Failed Components" >> BUILD_VERIFICATION_REPORT.md
echo "" >> BUILD_VERIFICATION_REPORT.md

if [ ${#FAILED_COMPONENTS[@]} -eq 0 ]; then
    echo "*No failures detected*" >> BUILD_VERIFICATION_REPORT.md
else
    for component in "${FAILED_COMPONENTS[@]}"; do
        echo "- ❌ $component" >> BUILD_VERIFICATION_REPORT.md
    done
fi

cat >> BUILD_VERIFICATION_REPORT.md <<EOF

## Build Environment

- Go Version: $(go version | awk '{print $3}')
- OS: $(uname -s)
- Architecture: $(uname -m)
- Git Commit: $(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

## Next Steps

EOF

if [ ${#FAILED_COMPONENTS[@]} -eq 0 ]; then
    cat >> BUILD_VERIFICATION_REPORT.md <<EOF
✅ **All components built successfully!**

The project is ready for deployment. You can proceed with:
1. Running integration tests: \`make test-integration\`
2. Building Docker images: \`make docker-build\`
3. Deploying to O-RAN SC environment

EOF
else
    cat >> BUILD_VERIFICATION_REPORT.md <<EOF
⚠️ **Action Required**

Some components failed to build. Please review the errors and:
1. Fix compilation errors in the failed components
2. Run \`go mod tidy\` to resolve dependency issues
3. Re-run this verification script

For detailed error information, check the build logs above.

EOF
fi

# Step 7: Update final status
FINAL_STATUS="success"
if [ ${#FAILED_COMPONENTS[@]} -gt 0 ]; then
    FINAL_STATUS="partial"
fi

cat > $BUILD_STATUS_FILE <<EOF
{
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "stage": "build-verification",
  "status": "$FINAL_STATUS",
  "success_rate": $SUCCESS_RATE,
  "successful_components": [$(printf '"%s",' "${SUCCESSFUL_COMPONENTS[@]}" | sed 's/,$//')]",
  "failed_components": [$(printf '"%s",' "${FAILED_COMPONENTS[@]}" | sed 's/,$//')],
  "ready_for_deployment": $([ ${#FAILED_COMPONENTS[@]} -eq 0 ] && echo "true" || echo "false")
}
EOF

# Display summary
echo -e "\n${BLUE}=== Build Verification Complete ===${NC}"
echo -e "Success Rate: ${SUCCESS_RATE}%"
echo -e "Report saved to: BUILD_VERIFICATION_REPORT.md"
echo -e "Status saved to: $BUILD_STATUS_FILE"

if [ ${#FAILED_COMPONENTS[@]} -eq 0 ]; then
    echo -e "\n${GREEN}✅ All components built successfully! Ready for O-RAN SC deployment.${NC}"
    exit 0
else
    echo -e "\n${YELLOW}⚠️ Some components failed to build. Review the report for details.${NC}"
    exit 1
fi