# Project Structure

## Root Directory Layout

```
├── cmd/                    # Go application entry points
│   └── xapp-hello-world/   # Example xApp implementation
├── ui/                     # React frontend application
├── helm/                   # Helm charts for deployment
├── ric-dep/               # RIC deployment dependencies (submodule)
├── scripts/               # Automation and utility scripts
├── build/                 # Build artifacts and Docker contexts
├── .github/               # GitHub Actions CI/CD workflows
├── .kiro/                 # Kiro AI assistant configuration
└── .vscode/               # VS Code workspace settings
```

## Go Backend Structure

### `/cmd` Directory
- Contains main application entry points
- Each subdirectory represents a deployable service
- Follow Go project layout conventions
- Example: `cmd/xapp-hello-world/main.go`

### Go Module Organization
- Root `go.mod` defines the main module: `github.com/oran/near-rt-ric-new`
- Uses Go 1.21 features
- Dependencies focused on O-RAN and gRPC libraries

## Frontend Structure

### `/ui` Directory
- Standard Create React App structure
- `src/` contains React components and application logic
- `public/` contains static assets
- `package.json` defines Node.js dependencies
- Follows React best practices and conventions

## Deployment Structure

### `/helm` Directory
- Contains Helm charts for Kubernetes deployment
- `chartmuseum/` for local chart repository
- `xapp-hello-world/` example xApp chart with:
  - `Chart.yaml` - Chart metadata
  - `values.yaml` - Default configuration
  - `templates/` - Kubernetes manifests

### `/ric-dep` Directory
- Git submodule containing RIC deployment dependencies
- Based on O-RAN SC components
- Contains production-grade SMO stack configurations

## Scripts and Automation

### `/scripts` Directory
- `health-check.sh` - Deployment verification script
- Additional automation scripts for setup and maintenance

## Build and CI/CD

### `/build` Directory
- Docker build contexts
- Build artifacts and configurations

### `/.github` Directory
- GitHub Actions workflows
- CI/CD pipeline definitions

## Configuration Files

### Root Level Files
- `Makefile` - Build automation and common tasks
- `Dockerfile` - Container build instructions
- `go.mod/go.sum` - Go dependency management
- `README.md` - Project documentation
- `CHANGELOG.md` - Version history
- `LICENSE` - Project license

## Development Guidelines

### File Naming Conventions
- Go files: lowercase with underscores (`main.go`, `health_check.go`)
- React components: PascalCase (`App.js`, `Dashboard.js`)
- Helm charts: lowercase with hyphens (`xapp-hello-world`)
- Scripts: lowercase with hyphens (`health-check.sh`)

### Directory Organization
- Keep related functionality grouped together
- Separate concerns between backend (`cmd/`) and frontend (`ui/`)
- Use standard Go project layout for backend services
- Follow React/Node.js conventions for frontend
- Maintain clear separation between source code and deployment artifacts

### Import Paths
- Use full module path for Go imports: `github.com/oran/near-rt-ric-new/...`
- Organize imports: standard library, third-party, local packages
- Use relative imports sparingly in React components