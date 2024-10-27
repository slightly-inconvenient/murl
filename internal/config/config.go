package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type RouteEnvironment struct {
	Allowlist []string `yaml:"allowlist" json:"allowlist"`
}

type RouteCheck struct {
	// Expr is a CEL expression evaluated against the request.
	// If the expression evaluates to true, the check passes.
	// If the expression evaluates to anything else, the check fails.
	Expr string `yaml:"expr" json:"expr"`

	// Error is the error message to return if the check fails.
	Error string `yaml:"error" json:"error"`
}

type RouteRedirect struct {
	// URL is the template to build the redirect URL from.
	URL string `yaml:"url" json:"url"`
}

type RouteDocumentation struct {
	// Title is a human-readable title for the route.
	Title string `yaml:"title" json:"title"`

	// Description is a human-readable description of the route.
	Description string `yaml:"description" json:"description"`
}

type Route struct {
	Path          string             `yaml:"path" json:"path"`
	Aliases       []string           `yaml:"aliases" json:"aliases"`
	Documentation RouteDocumentation `yaml:"documentation" json:"documentation"`
	Environment   RouteEnvironment   `yaml:"environment" json:"environment"`
	Params        map[string]string  `yaml:"params" json:"params"`
	Checks        []RouteCheck       `yaml:"checks" json:"checks"`
	Redirect      RouteRedirect      `yaml:"redirect" json:"redirect"`
}

type ServerTLSConfig struct {
	Cert string `yaml:"cert" json:"cert"`
	Key  string `yaml:"key" json:"key"`
}

type ServerTemplatesConfig struct {
	Root string `yaml:"root" json:"root"`
}

type ServerDocumentationConfig struct {
	Path      string                `yaml:"path" json:"path"`
	Templates ServerTemplatesConfig `yaml:"templates" json:"templates"`
}

type Server struct {
	Address       string                    `yaml:"address" json:"address"`
	TLS           ServerTLSConfig           `yaml:"tls" json:"tls"`
	Documentation ServerDocumentationConfig `yaml:"documentation" json:"documentation"`
}

type Config struct {
	Server Server  `yaml:"server" json:"server"`
	Routes []Route `yaml:"routes" json:"routes"`
}

func ParseConfigFile(path string) (Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("failed to read configuration file at %s: %w", path, err)
	}

	config := Config{}
	switch filepath.Ext(path) {
	case ".json":
		decoder := json.NewDecoder(bytes.NewReader(content))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&config); err != nil {
			return Config{}, fmt.Errorf("failed to parse configuration file: %w", err)
		}
	case ".yaml", ".yml":
		decoder := yaml.NewDecoder(bytes.NewReader(content))
		decoder.KnownFields(true)
		if err := decoder.Decode(&config); err != nil {
			return Config{}, fmt.Errorf("failed to parse configuration file: %w", err)
		}
	default:
		return Config{}, fmt.Errorf("unsupported configuration file extension: %q (supported are .yaml, .yml and .json)", filepath.Ext(path))
	}

	return config, nil
}
