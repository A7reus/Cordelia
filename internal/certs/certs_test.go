package certs

import (
	"crypto/tls"
	"crypto/x509"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOrCreateCreatesAndReuses(t *testing.T) {
	dir := t.TempDir()

	c1, err := LoadOrCreate(dir)
	if err != nil {
		t.Fatalf("first load: %v", err)
	}
	if len(c1.Certificate) == 0 {
		t.Fatalf("no certificate")
	}
	fp1, err := Fingerprint(c1)
	if err != nil {
		t.Fatalf("fingerprint: %v", err)
	}
	if len(fp1) != 64 {
		t.Fatalf("expected 64 hex chars, got %d", len(fp1))
	}

	c2, err := LoadOrCreate(dir)
	if err != nil {
		t.Fatalf("second load: %v", err)
	}
	fp2, err := Fingerprint(c2)
	if err != nil {
		t.Fatalf("fingerprint2: %v", err)
	}
	if fp1 != fp2 {
		t.Fatalf("fingerprint mismatch %s vs %s", fp1, fp2)
	}

	certPath := filepath.Join(dir, "cordelia", "cert.pem")
	keyPath := filepath.Join(dir, "cordelia", "key.pem")
	if fi, err := os.Stat(certPath); err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("cert perm want 600 got %v err %v", fi.Mode().Perm(), err)
	}
	if fi, err := os.Stat(keyPath); err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("key perm want 600")
	}

	if _, err := x509.ParseCertificate(c1.Certificate[0]); err != nil {
		t.Fatalf("parse cert: %v", err)
	}
}

func TestFingerprintEmpty(t *testing.T) {
	_, err := Fingerprint(tls.Certificate{})
	if err == nil {
		t.Fatalf("expected error for empty cert")
	}
}
