package server

import (
	"errors"
	"fmt"
	"os"
)

type InputTLSConfig struct {
	Cert string `yaml:"cert" json:"cert"`
	Key  string `yaml:"key" json:"key"`
}

type TLSConfig struct {
	cert string
	key  string
}

type InputConfig struct {
	Address string         `yaml:"address" json:"address"`
	TLS     InputTLSConfig `yaml:"tls" json:"tls"`
}

type Config struct {
	address string
	tls     TLSConfig
	valid   bool
}

// NewConfig parses the input server configuration and returns a validated server configuration.
func NewConfig(config InputConfig) (Config, error) {
	if config.Address == "" {
		return Config{}, fmt.Errorf("server address is required")
	}

	if config.TLS.Cert != "" {
		if config.TLS.Key == "" {
			return Config{}, fmt.Errorf("server TLS key is required when TLS cert is provided")
		}

		if _, err := os.Stat(config.TLS.Cert); errors.Is(err, os.ErrNotExist) {
			return Config{}, fmt.Errorf("server TLS cert file at path %q does not exist", config.TLS.Cert)
		}
	}

	if config.TLS.Key != "" {
		if config.TLS.Cert == "" {
			return Config{}, fmt.Errorf("server TLS cert is required when TLS key is provided")
		}

		if _, err := os.Stat(config.TLS.Key); errors.Is(err, os.ErrNotExist) {
			return Config{}, fmt.Errorf("server TLS key file at path %q does not exist", config.TLS.Key)
		}
	}

	return Config{
		address: config.Address,
		tls: TLSConfig{
			cert: config.TLS.Cert,
			key:  config.TLS.Key,
		},
		valid: true,
	}, nil
}
