package config

import (
	"os"

	"github.com/yaninyzwitty/chat/packages/shared/util"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Debug          bool           `yaml:"debug"`
	AuthPort       int            `yaml:"authPort"`
	UserPort       int            `yaml:"userPort"`
	MetricsPort1   int            `yaml:"metricsPort1"`
	MetricsPort2   int            `yaml:"metricsPort2"`
	DatabaseConfig DatabaseConfig `yaml:"db"`
}

type DatabaseConfig struct {
	Username string `yaml:"username"`
	// path to secure-connect.zip
	Path    string `yaml:"path"`
	Timeout int    `yaml:"timeout"`
}

// LoadConfig loads a YAML config file into the receiver.
func (c *Config) LoadConfig(path string) {
	// read the file by the path
	f, err := os.ReadFile(path)
	util.Fail(err, "failed to read config")

	err = yaml.Unmarshal(f, c)
	util.Fail(err, "failed to parse config")

}
