# Policy Management Framework

This document describes the implementation of the Policy Management Framework for the O-RAN Near-RT RIC platform, which provides comprehensive policy lifecycle management, validation, conflict detection, and distribution capabilities.

## Overview

The Policy Management Framework extends the basic A1 interface implementation with advanced features required for production-grade policy management in O-RAN environments. It addresses the requirements specified in task 3.2 of the O-RAN RIC rewrite specification.

## Key Features

### 1. JSON Schema Validation

The framework provides comprehensive JSON schema validation for both policy types and policy instances:

- **Policy Type Schema Validation**: Validates that policy type schemas are valid JSON Schema documents
- **Policy Instance Validation**: Validates policy instances against their corresponding policy type schemas
- **Real-time Validation**: Validation occurs during policy creation and updates
- **Detailed Error Reporting**: Provides field-level validation errors with descriptive messages

### 2. Policy Instance Lifecycle Management

Complete lifecycle management for policy instances:

- **Create**: Create new policy instances with validation and conflict detection
- **Update**: Update existing policy instances with re-validation and conflict checking
- **Delete**: Delete policy instances with proper cleanup and xApp notification
- **Status Tracking**: Track policy instance status throughout its lifecycle

### 3. Policy Conflict Detection and Resolution

Advanced conflict detection mechanisms:

- **Resource Conflicts**: Detect when policies target the same resources
- **Parameter Conflicts**: Identify conflicting parameter values between policies
- **Priority Conflicts**: Handle policies with same priority but different actions
- **Exclusive Conflicts**: Manage exclusive policies that cannot coexist
- **Conflict Resolution**: Provide mechanisms to resolve detected conflicts

### 4. Policy Distribution to xApps

Automated policy distribution system:

- **xApp Registration**: Register xApps for policy distribution
- **Automatic Distribution**: Distribute policies to relevant xApps upon creation/update
- **Status Tracking**: Monitor distribution status for each xApp
- **Retry Logic**: Handle distribution failures with retry mechanisms
- **Policy Withdrawal**: Notify xApps when policies are deleted

### 5. Policy Compliance Monitoring and Reporting

Continuous compliance monitoring:

- **Compliance Checks**: Regular compliance verification with xApps
- **Violation Detection**: Identify policy violations and non-compliance
- **Compliance Reports**: Generate detailed compliance reports
- **Real-time Monitoring**: Continuous monitoring of policy adherence

## Architecture

### Core Components

#### PolicyManager
The central component that orchestrates all policy management operations:

```go
type PolicyManager struct {
    a1Client           *A1MediatorClient
    xappClients        map[string]*XAppClient
    policyTypes        map[PolicyTypeID]*PolicyType
    policyInstances    map[PolicyInstanceID]*PolicyInstance
    conflicts          map[string]*PolicyConflict
    distributionStatus map[PolicyInstanceID]map[string]*PolicyDistributionStatus
    complianceReports  map[PolicyInstanceID]map[string]*PolicyComplianceReport
    // ... other fields
}
```

#### Background Workers
- **Distribution Worker**: Handles asynchronous policy distribution to xApps
- **Compliance Worker**: Performs periodic compliance checks and monitoring

### Data Models

#### Policy Conflict
```go
type PolicyConflict struct {
    ConflictID          string
    PolicyInstanceID    PolicyInstanceID
    ConflictingPolicyID PolicyInstanceID
    ConflictType        string
    Description         string
    Resolution          string
    DetectedAt          time.Time
}
```

#### Policy Distribution Status
```go
type PolicyDistributionStatus struct {
    PolicyInstanceID PolicyInstanceID
    XAppID           string
    Status           string
    Message          string
    LastUpdate       time.Time
}
```

#### Policy Compliance Report
```go
type PolicyComplianceReport struct {
    PolicyInstanceID PolicyInstanceID
    XAppID           string
    ComplianceStatus string
    Violations       []string
    LastCheck        time.Time
}
```

## API Endpoints

The framework extends the existing A1 API with additional endpoints:

### Policy Validation
- `POST /api/v1/a1/policytypes/{policyTypeId}/validate` - Validate policy instance
- `POST /api/v1/a1/policytypes/{policyTypeId}/validate-schema` - Validate policy type schema

### Policy Management
- `GET /api/v1/policies/conflicts` - Get all policy conflicts
- `POST /api/v1/policies/conflicts/{conflictId}/resolve` - Resolve a policy conflict
- `GET /api/v1/policies/{policyInstanceId}/distribution` - Get policy distribution status
- `GET /api/v1/policies/{policyInstanceId}/compliance` - Get policy compliance reports

### xApp Management
- `GET /api/v1/xapps/registration` - List registered xApps
- `POST /api/v1/xapps/registration` - Register an xApp for policy distribution
- `DELETE /api/v1/xapps/registration/{xappId}` - Unregister an xApp

## Usage Examples

### 1. Policy Type Schema Validation

```go
policyManager := NewPolicyManager(a1Client)
schema := json.RawMessage(`{
    "type": "object",
    "properties": {
        "priority": {"type": "integer"},
        "action": {"type": "string"}
    },
    "required": ["priority", "action"]
}`)

result, err := policyManager.ValidatePolicyType("qos-policy", schema)
if err != nil || !result.IsValid {
    // Handle validation errors
}
```

### 2. Policy Instance Creation with Validation

```go
policy := json.RawMessage(`{
    "priority": 10,
    "action": "allow"
}`)

err := policyManager.CreatePolicyInstance(ctx, "qos-policy", "instance-1", policy)
if err != nil {
    // Handle creation error (validation, conflicts, etc.)
}
```

### 3. Conflict Detection and Resolution

```go
// Get all conflicts
conflicts := policyManager.GetPolicyConflicts()

// Resolve a conflict
err := policyManager.ResolveConflict(conflictID, "Higher priority policy takes precedence")
```

### 4. xApp Registration and Distribution Monitoring

```go
// Register xApp
policyManager.RegisterXApp("qos-xapp", "http://qos-xapp:8080")

// Check distribution status
status := policyManager.GetPolicyDistributionStatus("policy-instance-1")
for xappID, distributionStatus := range status {
    fmt.Printf("xApp %s: %s\n", xappID, distributionStatus.Status)
}
```

### 5. Compliance Monitoring

```go
// Get compliance reports
reports := policyManager.GetPolicyComplianceReports("policy-instance-1")
for xappID, report := range reports {
    if report.ComplianceStatus == "NON_COMPLIANT" {
        fmt.Printf("xApp %s has violations: %v\n", xappID, report.Violations)
    }
}
```

## Configuration

The Policy Management Framework is automatically initialized when the dashboard server starts, provided that an A1 Mediator client is available:

```go
// In server initialization
if a1Client := clients.GetA1MediatorClient(); a1Client != nil {
    policyManager = NewPolicyManager(a1Client)
}
```

## Dependencies

The framework requires the following dependencies:

- `github.com/xeipuuv/gojsonschema` - JSON Schema validation
- Standard Go libraries for HTTP, JSON, and concurrency
- Existing A1 Mediator client implementation

## Testing

The framework includes comprehensive tests covering:

- Policy type and instance validation
- Conflict detection algorithms
- xApp registration and distribution
- Compliance monitoring
- API endpoint functionality

Run tests with:
```bash
go test -v ./pkg/dashboard -run TestPolicyManager
```

## Integration with O-RAN Components

The Policy Management Framework integrates seamlessly with:

- **A1 Mediator**: Uses existing A1 interface for basic policy operations
- **xApp Framework**: Distributes policies to registered xApps
- **Dashboard UI**: Provides REST APIs for web interface integration
- **Monitoring Stack**: Exports metrics for observability

## Performance Considerations

- **Asynchronous Operations**: Policy distribution and compliance checking are performed asynchronously
- **Caching**: Policy types and instances are cached for fast access
- **Concurrent Processing**: Uses goroutines for parallel policy operations
- **Resource Management**: Proper cleanup of resources when policies are deleted

## Security

- **Validation**: All inputs are validated before processing
- **Error Handling**: Comprehensive error handling prevents system crashes
- **Resource Limits**: Bounded channels prevent resource exhaustion
- **Access Control**: Integrates with existing RBAC mechanisms

## Future Enhancements

Potential future enhancements include:

- **Policy Templates**: Support for policy templates and parameterization
- **Advanced Conflict Resolution**: Machine learning-based conflict resolution
- **Policy Analytics**: Advanced analytics and reporting capabilities
- **Multi-tenant Support**: Support for tenant-specific policies
- **Policy Versioning**: Version control for policy instances

## Conclusion

The Policy Management Framework provides a comprehensive solution for managing policies in O-RAN environments, addressing all requirements specified in the O-RAN RIC rewrite specification. It offers production-grade features including validation, conflict detection, distribution, and compliance monitoring, making it suitable for real-world O-RAN deployments.