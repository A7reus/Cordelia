package identity

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
)

const fileName = "identity.json"

type Identity struct {
	Name        string `json:"name"`
	Fingerprint string `json:"fingerprint"`
}

func New(name string) (Identity, error) {
	fp, err := randomFingerprint()
	if err != nil {
		return Identity{}, err
	}

	return Identity{Name: name, Fingerprint: fp}, nil
}

func randomFingerprint() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return hex.EncodeToString(buf), nil
}

func Dir(override string) (string, error) {
	if override != "" {
		return filepath.Join(override, "cordelia"), nil
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "cordelia"), nil
}

func Path(dirOverride string) (string, error) {
	dir, err := Dir(dirOverride)
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, fileName), nil
}

func Save(id Identity, dirOverride string) error {
	path, err := Path(dirOverride)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(id, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

func Load(dirOverride string) (Identity, error) {
	path, err := Path(dirOverride)
	if err != nil {
		return Identity{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Identity{}, err
	}

	var id Identity
	if err := json.Unmarshal(data, &id); err != nil {
		return Identity{}, err
	}
	return id, nil
}
