package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const configFile = "config.json"

type Config struct {
	Port   int    `json:"port"`
	OutDir string `json:"out_dir"`
	TTL    string `json:"ttl"`
}

func Default() Config {
	return Config{
		Port:   47777,
		OutDir: "",
		TTL:    "10s",
	}
}

func TTLOrDefault(c Config) time.Duration {
	if d, err := time.ParseDuration(c.TTL); err == nil && d > 0 {
		return d
	}

	return 10 * time.Second
}

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

func path(dataDir string) (string, error) {
	d, err := dir(dataDir)
	if err != nil {
		return "", err
	}

	return filepath.Join(d, configFile), nil
}

func Load(dataDir string) (Config, error) {
	cfg := Default()
	p, err := path(dataDir)
	if err != nil {
		return cfg, err
	}

	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}

		return cfg, err
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return Default(), err
	}
	if cfg.Port == 0 {
		cfg.Port = 47777
	}
	if cfg.TTL == "" {
		cfg.TTL = "10s"
	}

	return cfg, nil
}

func Save(dataDir string, cfg Config) error {
	p, err := path(dataDir)
	if err != nil {
		return err
	}

	d := filepath.Dir(p)
	if err := os.MkdirAll(d, 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o600)
}
