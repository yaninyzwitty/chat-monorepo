package config

import (
	"os"

	"gopkg.in/yaml.v3"

	"github.com/yaninyzwitty/chat/pkg/util"
)

type Config struct {
	Debug          bool           `yaml:"debug"`
	AuthPort       int            `yaml:"authPort"`
	UserPort       int            `yaml:"userPort"`
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
