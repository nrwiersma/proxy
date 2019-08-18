package proxy

import (
	"io"

	"gopkg.in/yaml.v2"
)

// ParseConfig parses the configuration in the reader.
func ParseConfig(r io.Reader) (*Config, error) {
	config := &Config{}
	err := yaml.NewDecoder(r).Decode(config)
	return config, err
}

// Config represents proxy configuration.
type Config struct {
	Server      ServiceOpts           `yaml:"server"`
	Entrypoints map[string]Entrypoint `yaml:"entrypoints"`
	Backends    map[string]Backend    `yaml:"backends"`
	Routes      map[string]Route      `yaml:"routes"`
}
