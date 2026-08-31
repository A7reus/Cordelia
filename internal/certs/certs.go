package certs

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

const certFile = "cert.pem"
const keyFile = "key.pem"

func dir(dataDir string) (string, error) {
	if dataDir != "" {
		return filepath.Join(dataDir, "cordelia"), nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(base, "cordelia"), nil
}

func certPath(dataDir string) (string, error) {
	d, err := dir(dataDir)
	if err != nil {
		return "", err
	}

	return filepath.Join(d, certFile), nil
}

func keyPath(dataDir string) (string, error) {
	d, err := dir(dataDir)
	if err != nil {
		return "", err
	}

	return filepath.Join(d, keyFile), nil
}

func Fingerprint(cert tls.Certificate) (string, error) {
	if len(cert.Certificate) == 0 {
		return "", fmt.Errorf("no certificate")
	}

	parsed, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256(parsed.Raw)
	return hex.EncodeToString(sum[:]), nil
}

func LoadOrCreate(dataDir string) (tls.Certificate, error) {
	cPath, err := certPath(dataDir)
	if err != nil {
		return tls.Certificate{}, err
	}
	kPath, err := keyPath(dataDir)
	if err != nil {
		return tls.Certificate{}, err
	}

	if _, err := os.Stat(cPath); err == nil {
		if _, err := os.Stat(kPath); err == nil {
			return tls.LoadX509KeyPair(cPath, kPath)
		}
	}
	return generateAndSave(dataDir)
}

func generateAndSave(dataDir string) (tls.Certificate, error) {
	d, err := dir(dataDir)
	if err != nil {
		return tls.Certificate{}, err
	}
	if err := os.MkdirAll(d, 0o700); err != nil {
		return tls.Certificate{}, err
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, err
	}

	tmpl := x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "cordelia"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		BasicConstraintsValid: true,
	}

	cPath, _ := certPath(dataDir)
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, err
	}
	certOut, err := os.OpenFile(cPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return tls.Certificate{}, err
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: der}); err != nil {
		certOut.Close()
		return tls.Certificate{}, err
	}
	certOut.Close()

	kPath, _ := keyPath(dataDir)
	keyDer, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return tls.Certificate{}, err
	}
	keyOut, err := os.OpenFile(kPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return tls.Certificate{}, err
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDer}); err != nil {
		keyOut.Close()
		return tls.Certificate{}, err
	}
	keyOut.Close()

	return tls.LoadX509KeyPair(cPath, kPath)
}
