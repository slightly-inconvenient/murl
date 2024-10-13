package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/slightly-inconvenient/murl"
	"github.com/slightly-inconvenient/murl/internal/server"
	"gopkg.in/yaml.v3"
)

type inputConfig struct {
	Server server.InputConfig `yaml:"server" json:"server"`
	Routes []murl.InputRoute  `yaml:"routes" json:"routes"`
}

func ParseConfigFile(path string) (server.Config, []murl.Route, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return server.Config{}, nil, fmt.Errorf("failed to read configuration file at %s: %w", path, err)
	}

	config := inputConfig{}
	switch filepath.Ext(path) {
	case ".json":
		decoder := json.NewDecoder(bytes.NewReader(content))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&config); err != nil {
			return server.Config{}, nil, fmt.Errorf("failed to parse configuration file: %w", err)
		}
	case ".yaml", ".yml":
		decoder := yaml.NewDecoder(bytes.NewReader(content))
		decoder.KnownFields(true)
		if err := decoder.Decode(&config); err != nil {
			return server.Config{}, nil, fmt.Errorf("failed to parse configuration file: %w", err)
		}
	default:
		return server.Config{}, nil, fmt.Errorf("unsupported configuration file extension: %q (supported are .yaml, .yml and .json)", filepath.Ext(path))
	}

	srv, err := server.NewConfig(config.Server)
	if err != nil {
		return server.Config{}, nil, err
	}

	routes, err := murl.NewRoutes(config.Routes)
	if err != nil {
		return server.Config{}, nil, err
	}

	return srv, routes, nil
}
