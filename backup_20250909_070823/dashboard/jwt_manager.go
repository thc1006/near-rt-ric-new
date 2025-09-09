/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTManager handles JWT token operations
type JWTManager struct {
	privateKey    *rsa.PrivateKey
	publicKey     *rsa.PublicKey
	issuer        string
	tokenDuration time.Duration
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(issuer string, tokenDuration time.Duration) (*JWTManager, error) {
	// Generate RSA key pair for JWT signing
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key pair: %w", err)
	}

	return &JWTManager{
		privateKey:    privateKey,
		publicKey:     &privateKey.PublicKey,
		issuer:        issuer,
		tokenDuration: tokenDuration,
	}, nil
}

// NewJWTManagerWithKeys creates a JWT manager with provided keys
func NewJWTManagerWithKeys(privateKeyPEM, publicKeyPEM []byte, issuer string, tokenDuration time.Duration) (*JWTManager, error) {
	// Parse private key
	privateKeyBlock, _ := pem.Decode(privateKeyPEM)
	if privateKeyBlock == nil {
		return nil, fmt.Errorf("failed to decode private key PEM")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// Parse public key
	publicKeyBlock, _ := pem.Decode(publicKeyPEM)
	if publicKeyBlock == nil {
		return nil, fmt.Errorf("failed to decode public key PEM")
	}

	publicKeyInterface, err := x509.ParsePKIXPublicKey(publicKeyBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	publicKey, ok := publicKeyInterface.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not RSA")
	}

	return &JWTManager{
		privateKey:    privateKey,
		publicKey:     publicKey,
		issuer:        issuer,
		tokenDuration: tokenDuration,
	}, nil
}

// GenerateToken generates a JWT token for a user
func (jm *JWTManager) GenerateToken(user *User, sessionID string) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(jm.tokenDuration)

	// Extract permissions from user roles
	permissions := jm.extractPermissions(user.Roles)

	claims := &JWTClaims{
		UserID:      user.ID,
		Username:    user.Username,
		Roles:       user.Roles,
		Permissions: permissions,
		SessionID:   sessionID,
		IssuedAt:    now.Unix(),
		ExpiresAt:   expiresAt.Unix(),
		Issuer:      jm.issuer,
		Subject:     user.ID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(jm.privateKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, expiresAt, nil
}

// GenerateServiceAccountToken generates a JWT token for a service account
func (jm *JWTManager) GenerateServiceAccountToken(serviceAccount *ServiceAccount) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour) // Service account tokens last 24 hours

	// Extract permissions from service account roles
	permissions := jm.extractPermissions(serviceAccount.Roles)

	claims := &JWTClaims{
		UserID:      serviceAccount.ID,
		Username:    serviceAccount.Name,
		Roles:       serviceAccount.Roles,
		Permissions: permissions,
		SessionID:   fmt.Sprintf("sa-%s", serviceAccount.ID),
		IssuedAt:    now.Unix(),
		ExpiresAt:   expiresAt.Unix(),
		Issuer:      jm.issuer,
		Subject:     serviceAccount.ID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(jm.privateKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign service account token: %w", err)
	}

	return tokenString, expiresAt, nil
}

// ValidateToken validates and parses a JWT token
func (jm *JWTManager) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jm.publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Check if token is expired
	if time.Now().Unix() > claims.ExpiresAt {
		return nil, fmt.Errorf("token is expired")
	}

	return claims, nil
}

// RefreshToken generates a new token with extended expiration
func (jm *JWTManager) RefreshToken(tokenString string) (string, time.Time, error) {
	claims, err := jm.ValidateToken(tokenString)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("invalid token for refresh: %w", err)
	}

	// Check if token is close to expiration (within 1 hour)
	if time.Now().Unix() > claims.ExpiresAt-3600 {
		return "", time.Time{}, fmt.Errorf("token is too close to expiration for refresh")
	}

	now := time.Now()
	expiresAt := now.Add(jm.tokenDuration)

	newClaims := &JWTClaims{
		UserID:      claims.UserID,
		Username:    claims.Username,
		Roles:       claims.Roles,
		Permissions: claims.Permissions,
		SessionID:   claims.SessionID,
		IssuedAt:    now.Unix(),
		ExpiresAt:   expiresAt.Unix(),
		Issuer:      jm.issuer,
		Subject:     claims.Subject,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, newClaims)
	newTokenString, err := token.SignedString(jm.privateKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign refreshed token: %w", err)
	}

	return newTokenString, expiresAt, nil
}

// GetPublicKeyPEM returns the public key in PEM format
func (jm *JWTManager) GetPublicKeyPEM() ([]byte, error) {
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(jm.publicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal public key: %w", err)
	}

	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	return publicKeyPEM, nil
}

// GetPrivateKeyPEM returns the private key in PEM format
func (jm *JWTManager) GetPrivateKeyPEM() ([]byte, error) {
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(jm.privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	return privateKeyPEM, nil
}

// extractPermissions extracts permissions from roles
func (jm *JWTManager) extractPermissions(roleNames []string) []string {
	permissionSet := make(map[string]bool)
	
	for _, roleName := range roleNames {
		for _, systemRole := range SystemRoles {
			if systemRole.ID == roleName || systemRole.Name == roleName {
				for _, permission := range systemRole.Permissions {
					permissionSet[permission.ID] = true
				}
				break
			}
		}
	}

	permissions := make([]string, 0, len(permissionSet))
	for permission := range permissionSet {
		permissions = append(permissions, permission)
	}

	return permissions
}

// Custom JWT claims that implement jwt.Claims interface
func (c JWTClaims) Valid() error {
	now := time.Now().Unix()
	
	if c.ExpiresAt < now {
		return fmt.Errorf("token is expired")
	}
	
	if c.IssuedAt > now {
		return fmt.Errorf("token used before issued")
	}
	
	return nil
}