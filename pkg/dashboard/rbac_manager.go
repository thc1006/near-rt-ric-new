/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"fmt"
	"strings"
	"sync"
)

// RBACManager handles role-based access control
type RBACManager struct {
	roles       map[string]*Role
	permissions map[string]*Permission
	mutex       sync.RWMutex
}

// NewRBACManager creates a new RBAC manager
func NewRBACManager() *RBACManager {
	rbac := &RBACManager{
		roles:       make(map[string]*Role),
		permissions: make(map[string]*Permission),
	}

	// Initialize with system roles
	rbac.initializeSystemRoles()
	
	return rbac
}

// initializeSystemRoles initializes the system with predefined roles
func (rbac *RBACManager) initializeSystemRoles() {
	for _, role := range SystemRoles {
		rbac.roles[role.ID] = &role
		
		// Add permissions to the permissions map
		for _, permission := range role.Permissions {
			rbac.permissions[permission.ID] = &permission
		}
	}
}

// CreateRole creates a new role
func (rbac *RBACManager) CreateRole(role *Role) error {
	rbac.mutex.Lock()
	defer rbac.mutex.Unlock()

	if _, exists := rbac.roles[role.ID]; exists {
		return fmt.Errorf("role with ID %s already exists", role.ID)
	}

	rbac.roles[role.ID] = role
	
	// Add permissions to the permissions map
	for _, permission := range role.Permissions {
		rbac.permissions[permission.ID] = &permission
	}

	return nil
}

// UpdateRole updates an existing role
func (rbac *RBACManager) UpdateRole(roleID string, updates *Role) error {
	rbac.mutex.Lock()
	defer rbac.mutex.Unlock()

	role, exists := rbac.roles[roleID]
	if !exists {
		return fmt.Errorf("role with ID %s not found", roleID)
	}

	if role.IsSystem {
		return fmt.Errorf("cannot update system role %s", roleID)
	}

	// Update role fields
	if updates.Name != "" {
		role.Name = updates.Name
	}
	if updates.Description != "" {
		role.Description = updates.Description
	}
	if updates.Permissions != nil {
		role.Permissions = updates.Permissions
		
		// Update permissions map
		for _, permission := range updates.Permissions {
			rbac.permissions[permission.ID] = &permission
		}
	}

	return nil
}

// DeleteRole deletes a role
func (rbac *RBACManager) DeleteRole(roleID string) error {
	rbac.mutex.Lock()
	defer rbac.mutex.Unlock()

	role, exists := rbac.roles[roleID]
	if !exists {
		return fmt.Errorf("role with ID %s not found", roleID)
	}

	if role.IsSystem {
		return fmt.Errorf("cannot delete system role %s", roleID)
	}

	delete(rbac.roles, roleID)
	return nil
}

// GetRole retrieves a role by ID
func (rbac *RBACManager) GetRole(roleID string) (*Role, error) {
	rbac.mutex.RLock()
	defer rbac.mutex.RUnlock()

	role, exists := rbac.roles[roleID]
	if !exists {
		return nil, fmt.Errorf("role with ID %s not found", roleID)
	}

	return role, nil
}

// GetAllRoles retrieves all roles
func (rbac *RBACManager) GetAllRoles() []*Role {
	rbac.mutex.RLock()
	defer rbac.mutex.RUnlock()

	roles := make([]*Role, 0, len(rbac.roles))
	for _, role := range rbac.roles {
		roles = append(roles, role)
	}

	return roles
}

// CheckPermission checks if a user has permission to perform an action on a resource
func (rbac *RBACManager) CheckPermission(userRoles []string, resource, action string) bool {
	rbac.mutex.RLock()
	defer rbac.mutex.RUnlock()

	for _, roleName := range userRoles {
		role, exists := rbac.roles[roleName]
		if !exists {
			continue
		}

		for _, permission := range role.Permissions {
			if rbac.matchesPermission(permission, resource, action) {
				return true
			}
		}
	}

	return false
}

// CheckPermissionWithClaims checks permission using JWT claims
func (rbac *RBACManager) CheckPermissionWithClaims(claims *JWTClaims, resource, action string) bool {
	// Check direct permissions first
	for _, permissionID := range claims.Permissions {
		permission, exists := rbac.permissions[permissionID]
		if exists && rbac.matchesPermission(*permission, resource, action) {
			return true
		}
	}

	// Check role-based permissions
	return rbac.CheckPermission(claims.Roles, resource, action)
}

// matchesPermission checks if a permission matches the requested resource and action
func (rbac *RBACManager) matchesPermission(permission Permission, resource, action string) bool {
	// Check for wildcard permissions
	if permission.Resource == "*" && permission.Action == "*" {
		return true
	}

	// Check resource match
	resourceMatch := permission.Resource == "*" || permission.Resource == resource || 
		rbac.matchesPattern(permission.Resource, resource)

	// Check action match
	actionMatch := permission.Action == "*" || permission.Action == action ||
		rbac.matchesPattern(permission.Action, action)

	return resourceMatch && actionMatch
}

// matchesPattern checks if a pattern matches a value (supports wildcards)
func (rbac *RBACManager) matchesPattern(pattern, value string) bool {
	if pattern == "*" {
		return true
	}

	// Handle prefix wildcards (e.g., "e2:*" matches "e2:read", "e2:write")
	if strings.HasSuffix(pattern, ":*") {
		prefix := strings.TrimSuffix(pattern, ":*")
		return strings.HasPrefix(value, prefix+":")
	}

	// Handle suffix wildcards (e.g., "*:read" matches "e2:read", "a1:read")
	if strings.HasPrefix(pattern, "*:") {
		suffix := strings.TrimPrefix(pattern, "*:")
		return strings.HasSuffix(value, ":"+suffix)
	}

	return pattern == value
}

// GetUserPermissions returns all permissions for a user based on their roles
func (rbac *RBACManager) GetUserPermissions(userRoles []string) []Permission {
	rbac.mutex.RLock()
	defer rbac.mutex.RUnlock()

	permissionSet := make(map[string]Permission)

	for _, roleName := range userRoles {
		role, exists := rbac.roles[roleName]
		if !exists {
			continue
		}

		for _, permission := range role.Permissions {
			permissionSet[permission.ID] = permission
		}
	}

	permissions := make([]Permission, 0, len(permissionSet))
	for _, permission := range permissionSet {
		permissions = append(permissions, permission)
	}

	return permissions
}

// ValidateRoleAssignment validates if roles can be assigned to a user
func (rbac *RBACManager) ValidateRoleAssignment(roleIDs []string) error {
	rbac.mutex.RLock()
	defer rbac.mutex.RUnlock()

	for _, roleID := range roleIDs {
		if _, exists := rbac.roles[roleID]; !exists {
			return fmt.Errorf("role with ID %s does not exist", roleID)
		}
	}

	return nil
}

// GetRoleHierarchy returns the role hierarchy for display purposes
func (rbac *RBACManager) GetRoleHierarchy() map[string][]string {
	rbac.mutex.RLock()
	defer rbac.mutex.RUnlock()

	hierarchy := make(map[string][]string)

	// Simple hierarchy based on permission count and system roles
	systemRoles := []string{}
	customRoles := []string{}

	for roleID, role := range rbac.roles {
		if role.IsSystem {
			systemRoles = append(systemRoles, roleID)
		} else {
			customRoles = append(customRoles, roleID)
		}
	}

	hierarchy["system"] = systemRoles
	hierarchy["custom"] = customRoles

	return hierarchy
}

// CreatePermission creates a new permission
func (rbac *RBACManager) CreatePermission(permission *Permission) error {
	rbac.mutex.Lock()
	defer rbac.mutex.Unlock()

	if _, exists := rbac.permissions[permission.ID]; exists {
		return fmt.Errorf("permission with ID %s already exists", permission.ID)
	}

	rbac.permissions[permission.ID] = permission
	return nil
}

// GetAllPermissions returns all available permissions
func (rbac *RBACManager) GetAllPermissions() []Permission {
	rbac.mutex.RLock()
	defer rbac.mutex.RUnlock()

	permissions := make([]Permission, 0, len(rbac.permissions))
	for _, permission := range rbac.permissions {
		permissions = append(permissions, *permission)
	}

	return permissions
}

// GetPermissionsByResource returns permissions for a specific resource
func (rbac *RBACManager) GetPermissionsByResource(resource string) []Permission {
	rbac.mutex.RLock()
	defer rbac.mutex.RUnlock()

	var permissions []Permission
	for _, permission := range rbac.permissions {
		if permission.Resource == resource || permission.Resource == "*" {
			permissions = append(permissions, *permission)
		}
	}

	return permissions
}