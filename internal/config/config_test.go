package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefault(t *testing.T) {
	c := Default()

	if c.Port != 47777 {
		t.Fatalf("expected port 47777, got %d", c.Port)
	}
	if c.TTL != "10s" {
		t.Fatalf("expected ttl 10s, got %q", c.TTL)
	}
	if TTLOrDefault(c) != 10*time.Second {
		t.Fatalf("expected 10s duration")
	}
}

func TestTTLOrDefaultFallback(t *testing.T) {
	c := Config{TTL: "bad"}
	if TTLOrDefault(c) != 10*time.Second {
		t.Fatalf("expected fallback 10s for bad ttl")
	}

	c.TTL = "5s"
	if TTLOrDefault(c) != 5*time.Second {
		t.Fatalf("expected 5s")
	}
}

func TestLoadNoFile(t *testing.T) {
	dir := t.TempDir()
	c, err := Load(dir)

	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if c.Port != 47777 {
		t.Fatalf("expected default port")
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	want := Config{Port: 5000, OutDir: "/tmp/custom", TTL: "5s"}
	if err := Save(dir, want); err != nil {
		t.Fatalf("save: %v", err)
	}

	fi, err := os.Stat(filepath.Join(dir, "cordelia", "config.json"))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("expected 600, got %o", fi.Mode().Perm())
	}

	got, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.Port != want.Port || got.OutDir != want.OutDir || got.TTL != want.TTL {
		t.Fatalf("mismatch got %+v want %+v", got, want)
	}
}
