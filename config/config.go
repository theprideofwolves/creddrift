package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ScanTargets []string `yaml:"scan_targets"`
	IgnoreDirs  []string `yaml:"ignore_dirs"`
}

func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cfg Config
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, err
	}

	if len(cfg.ScanTargets) == 0 {
		cfg.ScanTargets = []string{"."}
	}

	return &cfg, nil
}
