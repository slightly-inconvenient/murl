package testtls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

type generateOptions struct {
	dnsAddresses []string
	ipAddresses  []net.IP
	notBefore    time.Time
	notAfter     time.Time

	serverCertOrganizationName string
	serverCertCommonName       string
}

type certificate struct {
	Key                *ecdsa.PrivateKey
	Certificate        *x509.Certificate
	CertificateEncoded []byte
}

func CreateTestTLSCertificates(tmpDir string) (string, string, string) {
	caFile := tmpDir + "/ca.crt"
	certFile := tmpDir + "/tls.crt"
	keyFile := tmpDir + "/tls.key"

	options := &generateOptions{
		dnsAddresses:               []string{"localhost"},
		ipAddresses:                []net.IP{net.ParseIP("127.0.0.1")},
		notBefore:                  time.Now(),
		notAfter:                   time.Now().Add(365 * 24 * time.Hour),
		serverCertOrganizationName: "Test",
		serverCertCommonName:       "test_server_cert",
	}

	if err := generateCertificates(caFile, certFile, keyFile, options); err != nil {
		panic(fmt.Errorf("failed to generate certificates: %w", err))
	}

	return caFile, certFile, keyFile
}

func generateCertificates(caFile string, certFile string, keyFile string, options *generateOptions) error {
	caCert, err := generateCA(options)
	if err != nil {
		return fmt.Errorf("failed to generate ca: %w", err)
	}

	cert, err := generateServerCertificate(options, caCert.Key, caCert.Certificate)
	if err != nil {
		return fmt.Errorf("failed to generate server certs: %w", err)
	}

	if err := writePrivateKey(keyFile, cert.Key); err != nil {
		return fmt.Errorf("failed to write private key to file: %w", err)
	}

	for filename, certBytes := range map[string][]byte{
		certFile: cert.CertificateEncoded,
		caFile:   caCert.CertificateEncoded,
	} {
		if err := writeCertitificate(filename, certBytes); err != nil {
			return fmt.Errorf("failed to write cert to file: %w", err)
		}
	}

	return nil
}

var currentSerialNumber *big.Int

func generateSerialNumber() (*big.Int, error) {
	if currentSerialNumber == nil {
		serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 64)

		serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
		if err != nil {
			return nil, fmt.Errorf("failed to generate seed serial number: %w", err)
		}

		currentSerialNumber = serialNumber
	}

	return currentSerialNumber.Add(currentSerialNumber, big.NewInt(1)), nil
}

func generateCA(options *generateOptions) (*certificate, error) {
	cert := &x509.Certificate{
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
			CommonName:   "Root CA",
		},
		NotBefore: options.notBefore,
		NotAfter:  options.notAfter,
		KeyUsage:  x509.KeyUsageCertSign,
		// ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	return generateCertificate(cert, nil, nil)
}

func generateServerCertificate(options *generateOptions, caKey *ecdsa.PrivateKey, caCert *x509.Certificate) (*certificate, error) {
	cert := x509.Certificate{
		Subject: pkix.Name{
			Organization: []string{options.serverCertOrganizationName},
			CommonName:   options.serverCertCommonName,
		},
		NotBefore:             options.notBefore,
		NotAfter:              options.notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
		IPAddresses:           options.ipAddresses,
		DNSNames:              options.dnsAddresses,
	}

	return generateCertificate(&cert, caKey, caCert)
}

func generateCertificate(cert *x509.Certificate, rootKey *ecdsa.PrivateKey, parent *x509.Certificate) (*certificate, error) {
	serialNumber, err := generateSerialNumber()
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	cert.SerialNumber = serialNumber

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate certificate keys: %w", err)
	}

	if parent == nil {
		parent = cert
	}

	if rootKey == nil {
		rootKey = key
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, parent, &key.PublicKey, rootKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	return &certificate{
		Key:                key,
		Certificate:        cert,
		CertificateEncoded: certBytes,
	}, nil
}

func writePrivateKey(filename string, key *ecdsa.PrivateKey) error {
	if err := os.MkdirAll(filepath.Dir(filename), 0o600); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return fmt.Errorf("failed to marshal: %w", err)
	}

	if err := pem.Encode(file, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}); err != nil {
		return fmt.Errorf("failed to pem encode ec private key: %w", err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func writeCertitificate(filename string, certBytes []byte) error {
	if err := os.MkdirAll(filepath.Dir(filename), 0o600); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	if err := pem.Encode(file, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes}); err != nil {
		return fmt.Errorf("failed to pem encode certificate: %w", err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
