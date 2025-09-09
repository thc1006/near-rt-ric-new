/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TLSManager handles TLS configuration and certificate management
type TLSManager struct {
	config       *TLSConfig
	certificates map[string]*tls.Certificate
	caCert       *x509.Certificate
	caKey        *rsa.PrivateKey
	mutex        sync.RWMutex
}

// TLSConfig represents TLS configuration
type TLSConfig struct {
	CertDir           string        `json:"certDir"`
	CAKeyFile         string        `json:"caKeyFile"`
	CACertFile        string        `json:"caCertFile"`
	ServerKeyFile     string        `json:"serverKeyFile"`
	ServerCertFile    string        `json:"serverCertFile"`
	ClientKeyFile     string        `json:"clientKeyFile"`
	ClientCertFile    string        `json:"clientCertFile"`
	KeySize           int           `json:"keySize"`
	CertValidityDays  int           `json:"certValidityDays"`
	Organization      string        `json:"organization"`
	Country           string        `json:"country"`
	Province          string        `json:"province"`
	City              string        `json:"city"`
	AutoRotate        bool          `json:"autoRotate"`
	RotationThreshold time.Duration `json:"rotationThreshold"`
}

// CertificateInfo represents certificate information
type CertificateInfo struct {
	Subject      string    `json:"subject"`
	Issuer       string    `json:"issuer"`
	SerialNumber string    `json:"serialNumber"`
	NotBefore    time.Time `json:"notBefore"`
	NotAfter     time.Time `json:"notAfter"`
	DNSNames     []string  `json:"dnsNames"`
	IPAddresses  []string  `json:"ipAddresses"`
	KeyUsage     []string  `json:"keyUsage"`
	IsCA         bool      `json:"isCA"`
	IsExpired    bool      `json:"isExpired"`
	DaysToExpiry int       `json:"daysToExpiry"`
}

// NewTLSManager creates a new TLS manager
func NewTLSManager(config *TLSConfig) (*TLSManager, error) {
	if config == nil {
		config = &TLSConfig{
			CertDir:           "./certs",
			CAKeyFile:         "ca-key.pem",
			CACertFile:        "ca-cert.pem",
			ServerKeyFile:     "server-key.pem",
			ServerCertFile:    "server-cert.pem",
			ClientKeyFile:     "client-key.pem",
			ClientCertFile:    "client-cert.pem",
			KeySize:           2048,
			CertValidityDays:  365,
			Organization:      "O-RAN SC",
			Country:           "US",
			Province:          "CA",
			City:              "San Francisco",
			AutoRotate:        true,
			RotationThreshold: 30 * 24 * time.Hour, // 30 days
		}
	}

	tm := &TLSManager{
		config:       config,
		certificates: make(map[string]*tls.Certificate),
	}

	// Create certificate directory if it doesn't exist
	if err := os.MkdirAll(config.CertDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create certificate directory: %w", err)
	}

	// Initialize CA certificate and key
	if err := tm.initializeCA(); err != nil {
		return nil, fmt.Errorf("failed to initialize CA: %w", err)
	}

	// Generate server certificates
	if err := tm.generateServerCertificate(); err != nil {
		return nil, fmt.Errorf("failed to generate server certificate: %w", err)
	}

	// Generate client certificates
	if err := tm.generateClientCertificate(); err != nil {
		return nil, fmt.Errorf("failed to generate client certificate: %w", err)
	}

	// Start certificate rotation routine if enabled
	if config.AutoRotate {
		go tm.certificateRotationRoutine()
	}

	return tm, nil
}

// initializeCA initializes the Certificate Authority
func (tm *TLSManager) initializeCA() error {
	caKeyPath := filepath.Join(tm.config.CertDir, tm.config.CAKeyFile)
	caCertPath := filepath.Join(tm.config.CertDir, tm.config.CACertFile)

	// Check if CA files already exist
	if _, err := os.Stat(caKeyPath); os.IsNotExist(err) {
		log.Println("Generating new CA certificate and key")
		return tm.generateCA()
	}

	// Load existing CA certificate and key
	log.Println("Loading existing CA certificate and key")
	return tm.loadCA()
}

// generateCA generates a new Certificate Authority
func (tm *TLSManager) generateCA() error {
	// Generate CA private key
	caKey, err := rsa.GenerateKey(rand.Reader, tm.config.KeySize)
	if err != nil {
		return fmt.Errorf("failed to generate CA private key: %w", err)
	}

	// Create CA certificate template
	caTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{tm.config.Organization},
			Country:       []string{tm.config.Country},
			Province:      []string{tm.config.Province},
			Locality:      []string{tm.config.City},
			CommonName:    "O-RAN RIC CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 0, tm.config.CertValidityDays*2), // CA valid for 2x server cert validity
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	// Create CA certificate
	caCertDER, err := x509.CreateCertificate(rand.Reader, &caTemplate, &caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("failed to create CA certificate: %w", err)
	}

	// Parse CA certificate
	caCert, err := x509.ParseCertificate(caCertDER)
	if err != nil {
		return fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	// Save CA private key
	caKeyPath := filepath.Join(tm.config.CertDir, tm.config.CAKeyFile)
	if err := tm.savePrivateKey(caKeyPath, caKey); err != nil {
		return fmt.Errorf("failed to save CA private key: %w", err)
	}

	// Save CA certificate
	caCertPath := filepath.Join(tm.config.CertDir, tm.config.CACertFile)
	if err := tm.saveCertificate(caCertPath, caCert); err != nil {
		return fmt.Errorf("failed to save CA certificate: %w", err)
	}

	tm.caCert = caCert
	tm.caKey = caKey

	log.Println("Generated new CA certificate and key")
	return nil
}

// loadCA loads existing Certificate Authority
func (tm *TLSManager) loadCA() error {
	// Load CA private key
	caKeyPath := filepath.Join(tm.config.CertDir, tm.config.CAKeyFile)
	caKey, err := tm.loadPrivateKey(caKeyPath)
	if err != nil {
		return fmt.Errorf("failed to load CA private key: %w", err)
	}

	// Load CA certificate
	caCertPath := filepath.Join(tm.config.CertDir, tm.config.CACertFile)
	caCert, err := tm.loadCertificate(caCertPath)
	if err != nil {
		return fmt.Errorf("failed to load CA certificate: %w", err)
	}

	tm.caCert = caCert
	tm.caKey = caKey

	log.Println("Loaded existing CA certificate and key")
	return nil
}

// generateServerCertificate generates server certificate
func (tm *TLSManager) generateServerCertificate() error {
	serverKeyPath := filepath.Join(tm.config.CertDir, tm.config.ServerKeyFile)
	serverCertPath := filepath.Join(tm.config.CertDir, tm.config.ServerCertFile)

	// Check if server certificate already exists and is valid
	if tm.isCertificateValid(serverCertPath) {
		log.Println("Server certificate is valid, skipping generation")
		return tm.loadServerCertificate()
	}

	log.Println("Generating new server certificate")

	// Generate server private key
	serverKey, err := rsa.GenerateKey(rand.Reader, tm.config.KeySize)
	if err != nil {
		return fmt.Errorf("failed to generate server private key: %w", err)
	}

	// Create server certificate template
	serverTemplate := x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization:  []string{tm.config.Organization},
			Country:       []string{tm.config.Country},
			Province:      []string{tm.config.Province},
			Locality:      []string{tm.config.City},
			CommonName:    "O-RAN RIC Server",
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(0, 0, tm.config.CertValidityDays),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		DNSNames:     []string{"localhost", "oran-ric-dashboard", "dashboard-api"},
	}

	// Create server certificate
	serverCertDER, err := x509.CreateCertificate(rand.Reader, &serverTemplate, tm.caCert, &serverKey.PublicKey, tm.caKey)
	if err != nil {
		return fmt.Errorf("failed to create server certificate: %w", err)
	}

	// Parse server certificate
	serverCert, err := x509.ParseCertificate(serverCertDER)
	if err != nil {
		return fmt.Errorf("failed to parse server certificate: %w", err)
	}

	// Save server private key
	if err := tm.savePrivateKey(serverKeyPath, serverKey); err != nil {
		return fmt.Errorf("failed to save server private key: %w", err)
	}

	// Save server certificate
	if err := tm.saveCertificate(serverCertPath, serverCert); err != nil {
		return fmt.Errorf("failed to save server certificate: %w", err)
	}

	// Load server certificate for TLS
	tlsCert, err := tls.LoadX509KeyPair(serverCertPath, serverKeyPath)
	if err != nil {
		return fmt.Errorf("failed to load server certificate for TLS: %w", err)
	}

	tm.mutex.Lock()
	tm.certificates["server"] = &tlsCert
	tm.mutex.Unlock()

	log.Println("Generated new server certificate")
	return nil
}

// generateClientCertificate generates client certificate
func (tm *TLSManager) generateClientCertificate() error {
	clientKeyPath := filepath.Join(tm.config.CertDir, tm.config.ClientKeyFile)
	clientCertPath := filepath.Join(tm.config.CertDir, tm.config.ClientCertFile)

	// Check if client certificate already exists and is valid
	if tm.isCertificateValid(clientCertPath) {
		log.Println("Client certificate is valid, skipping generation")
		return tm.loadClientCertificate()
	}

	log.Println("Generating new client certificate")

	// Generate client private key
	clientKey, err := rsa.GenerateKey(rand.Reader, tm.config.KeySize)
	if err != nil {
		return fmt.Errorf("failed to generate client private key: %w", err)
	}

	// Create client certificate template
	clientTemplate := x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject: pkix.Name{
			Organization:  []string{tm.config.Organization},
			Country:       []string{tm.config.Country},
			Province:      []string{tm.config.Province},
			Locality:      []string{tm.config.City},
			CommonName:    "O-RAN RIC Client",
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(0, 0, tm.config.CertValidityDays),
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	// Create client certificate
	clientCertDER, err := x509.CreateCertificate(rand.Reader, &clientTemplate, tm.caCert, &clientKey.PublicKey, tm.caKey)
	if err != nil {
		return fmt.Errorf("failed to create client certificate: %w", err)
	}

	// Parse client certificate
	clientCert, err := x509.ParseCertificate(clientCertDER)
	if err != nil {
		return fmt.Errorf("failed to parse client certificate: %w", err)
	}

	// Save client private key
	if err := tm.savePrivateKey(clientKeyPath, clientKey); err != nil {
		return fmt.Errorf("failed to save client private key: %w", err)
	}

	// Save client certificate
	if err := tm.saveCertificate(clientCertPath, clientCert); err != nil {
		return fmt.Errorf("failed to save client certificate: %w", err)
	}

	// Load client certificate for TLS
	tlsCert, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
	if err != nil {
		return fmt.Errorf("failed to load client certificate for TLS: %w", err)
	}

	tm.mutex.Lock()
	tm.certificates["client"] = &tlsCert
	tm.mutex.Unlock()

	log.Println("Generated new client certificate")
	return nil
}

// loadServerCertificate loads existing server certificate
func (tm *TLSManager) loadServerCertificate() error {
	serverKeyPath := filepath.Join(tm.config.CertDir, tm.config.ServerKeyFile)
	serverCertPath := filepath.Join(tm.config.CertDir, tm.config.ServerCertFile)

	tlsCert, err := tls.LoadX509KeyPair(serverCertPath, serverKeyPath)
	if err != nil {
		return fmt.Errorf("failed to load server certificate: %w", err)
	}

	tm.mutex.Lock()
	tm.certificates["server"] = &tlsCert
	tm.mutex.Unlock()

	return nil
}

// loadClientCertificate loads existing client certificate
func (tm *TLSManager) loadClientCertificate() error {
	clientKeyPath := filepath.Join(tm.config.CertDir, tm.config.ClientKeyFile)
	clientCertPath := filepath.Join(tm.config.CertDir, tm.config.ClientCertFile)

	tlsCert, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
	if err != nil {
		return fmt.Errorf("failed to load client certificate: %w", err)
	}

	tm.mutex.Lock()
	tm.certificates["client"] = &tlsCert
	tm.mutex.Unlock()

	return nil
}

// GetServerTLSConfig returns TLS configuration for server
func (tm *TLSManager) GetServerTLSConfig() *tls.Config {
	tm.mutex.RLock()
	serverCert := tm.certificates["server"]
	tm.mutex.RUnlock()

	if serverCert == nil {
		log.Println("Warning: Server certificate not available")
		return nil
	}

	// Create CA certificate pool
	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(tm.caCert)

	return &tls.Config{
		Certificates: []tls.Certificate{*serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caCertPool,
		MinVersion:   tls.VersionTLS13,
		MaxVersion:   tls.VersionTLS13,
		CipherSuites: []uint16{
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_AES_128_GCM_SHA256,
		},
	}
}

// GetClientTLSConfig returns TLS configuration for client
func (tm *TLSManager) GetClientTLSConfig() *tls.Config {
	tm.mutex.RLock()
	clientCert := tm.certificates["client"]
	tm.mutex.RUnlock()

	if clientCert == nil {
		log.Println("Warning: Client certificate not available")
		return nil
	}

	// Create CA certificate pool
	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(tm.caCert)

	return &tls.Config{
		Certificates: []tls.Certificate{*clientCert},
		RootCAs:      caCertPool,
		MinVersion:   tls.VersionTLS13,
		MaxVersion:   tls.VersionTLS13,
		CipherSuites: []uint16{
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_AES_128_GCM_SHA256,
		},
	}
}

// GetCertificateInfo returns information about a certificate
func (tm *TLSManager) GetCertificateInfo(certType string) (*CertificateInfo, error) {
	var certPath string
	switch certType {
	case "ca":
		certPath = filepath.Join(tm.config.CertDir, tm.config.CACertFile)
	case "server":
		certPath = filepath.Join(tm.config.CertDir, tm.config.ServerCertFile)
	case "client":
		certPath = filepath.Join(tm.config.CertDir, tm.config.ClientCertFile)
	default:
		return nil, fmt.Errorf("unknown certificate type: %s", certType)
	}

	cert, err := tm.loadCertificate(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate: %w", err)
	}

	return tm.extractCertificateInfo(cert), nil
}

// extractCertificateInfo extracts information from a certificate
func (tm *TLSManager) extractCertificateInfo(cert *x509.Certificate) *CertificateInfo {
	var keyUsage []string
	if cert.KeyUsage&x509.KeyUsageDigitalSignature != 0 {
		keyUsage = append(keyUsage, "Digital Signature")
	}
	if cert.KeyUsage&x509.KeyUsageKeyEncipherment != 0 {
		keyUsage = append(keyUsage, "Key Encipherment")
	}
	if cert.KeyUsage&x509.KeyUsageCertSign != 0 {
		keyUsage = append(keyUsage, "Certificate Sign")
	}

	var ipAddresses []string
	for _, ip := range cert.IPAddresses {
		ipAddresses = append(ipAddresses, ip.String())
	}

	daysToExpiry := int(time.Until(cert.NotAfter).Hours() / 24)

	return &CertificateInfo{
		Subject:      cert.Subject.String(),
		Issuer:       cert.Issuer.String(),
		SerialNumber: cert.SerialNumber.String(),
		NotBefore:    cert.NotBefore,
		NotAfter:     cert.NotAfter,
		DNSNames:     cert.DNSNames,
		IPAddresses:  ipAddresses,
		KeyUsage:     keyUsage,
		IsCA:         cert.IsCA,
		IsExpired:    time.Now().After(cert.NotAfter),
		DaysToExpiry: daysToExpiry,
	}
}

// certificateRotationRoutine handles automatic certificate rotation
func (tm *TLSManager) certificateRotationRoutine() {
	ticker := time.NewTicker(24 * time.Hour) // Check daily
	defer ticker.Stop()

	for range ticker.C {
		tm.checkAndRotateCertificates()
	}
}

// checkAndRotateCertificates checks and rotates certificates if needed
func (tm *TLSManager) checkAndRotateCertificates() {
	log.Println("Checking certificates for rotation")

	// Check server certificate
	if tm.needsRotation("server") {
		log.Println("Server certificate needs rotation")
		if err := tm.generateServerCertificate(); err != nil {
			log.Printf("Failed to rotate server certificate: %v", err)
		}
	}

	// Check client certificate
	if tm.needsRotation("client") {
		log.Println("Client certificate needs rotation")
		if err := tm.generateClientCertificate(); err != nil {
			log.Printf("Failed to rotate client certificate: %v", err)
		}
	}
}

// needsRotation checks if a certificate needs rotation
func (tm *TLSManager) needsRotation(certType string) bool {
	info, err := tm.GetCertificateInfo(certType)
	if err != nil {
		log.Printf("Failed to get certificate info for %s: %v", certType, err)
		return true // Assume rotation needed if we can't check
	}

	return time.Until(info.NotAfter) < tm.config.RotationThreshold
}

// isCertificateValid checks if a certificate file exists and is valid
func (tm *TLSManager) isCertificateValid(certPath string) bool {
	cert, err := tm.loadCertificate(certPath)
	if err != nil {
		return false
	}

	// Check if certificate is expired or will expire soon
	return time.Until(cert.NotAfter) > tm.config.RotationThreshold
}

// Helper methods for file operations

func (tm *TLSManager) savePrivateKey(path string, key *rsa.PrivateKey) error {
	keyFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer keyFile.Close()

	keyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}

	return pem.Encode(keyFile, keyPEM)
}

func (tm *TLSManager) loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	keyData, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	keyBlock, _ := pem.Decode(keyData)
	if keyBlock == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	return x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
}

func (tm *TLSManager) saveCertificate(path string, cert *x509.Certificate) error {
	certFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer certFile.Close()

	certPEM := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	}

	return pem.Encode(certFile, certPEM)
}

func (tm *TLSManager) loadCertificate(path string) (*x509.Certificate, error) {
	certData, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	certBlock, _ := pem.Decode(certData)
	if certBlock == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	return x509.ParseCertificate(certBlock.Bytes)
}