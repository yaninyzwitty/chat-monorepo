package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Debug          bool           `yaml:"debug"`
	AuthPort       int            `yaml:"authPort"`
	AuthClientPort int            `yaml:"authClientPort"`
	UserClientPort int            `yaml:"userClientPort"`
	UserPort       int            `yaml:"userPort"`
	MetricsPort1   int            `yaml:"metricsPort1"`
	MetricsPort2   int            `yaml:"metricsPort2"`
	DatabaseConfig DatabaseConfig `yaml:"db"`
}

type DatabaseConfig struct {
	Username string `yaml:"username"`
	// path to secure-connect.zip
	Path        string `yaml:"path"`
	Timeout     int    `yaml:"timeout"`
	Local_Host  string `yaml:"localHost"`
	LocalDBPort int    `yaml:"localDBPort"`
}

// LoadConfig loads a YAML config file into the receiver.
func (c *Config) LoadConfig(path string) error {
	// read the file by the path
	f, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	err = yaml.Unmarshal(f, c)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	return nil
}
