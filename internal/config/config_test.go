package config_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/slightly-inconvenient/murl/internal/config"
	"github.com/slightly-inconvenient/murl/internal/testtls"
)

func writeTempFile(t *testing.T, content string, file string) string {
	tempFile := filepath.Join(t.TempDir(), file)
	err := os.WriteFile(tempFile, []byte(content), 0o644)
	if err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}

	return tempFile
}

func writeConfig(t *testing.T, content string, suffix string) string {
	return writeTempFile(t, content, fmt.Sprintf("config.%s", suffix))
}

func writeConfigYAML(t *testing.T, content string) string {
	return writeConfig(t, content, "yaml")
}

func writeConfigJSON(t *testing.T, content string) string {
	return writeConfig(t, content, "json")
}

func TestConfig(t *testing.T) {
	t.Parallel()

	_, certPath, keyPath := testtls.CreateTestTLSCertificates(t.TempDir())

	tests := []struct {
		description   string
		configPath    string
		expectedError error
	}{
		{
			description: "valid YAML config",
			configPath: writeConfigYAML(t, fmt.Sprintf(`
server:
  address: ":8443"
  tls:
    cert: %s
    key: %s
routes:
- path: /example/{rest}
  aliases:
  - /example2/{rest}
  documentation: 
    title: "Example route"
    description: "An example route that picks params from the env, path, query, and headers"
  environment:
    allowlist:
      - EXAMPLE_HOST
  params:
    path: '{{.GetParam "rest"}}'
    host: '{{.GetEnv "EXAMPLE_HOST"}}'
    query: '{{.GetQuery "query"}}'
    contentType: '{{.GetHeader "content-type"}}'
  checks: 
  - expr: 'host != ""'
    error: "host is required"
  - expr: 'path != ""'
    error: "path is required"
  redirect:
    url: "https://{{.host}}/any/will/do/{{.path}}?q={{.query}}&h={{.contentType}}"
 `, certPath, keyPath)),
		},
		{
			description: "valid JSON config",
			configPath: writeConfigJSON(t, fmt.Sprintf(`{
  "server": {
    "address": ":8443",
    "tls": {
      "cert": %q,
      "key": %q
    }
  },
  "routes": [
    {
      "path": "/example/{rest}",
      "aliases": [
        "/example2/{rest}"
      ],
      "documentation": {
        "title": "Example route",
        "description": "An example route that picks params from the env, path, query, and headers"
      },
      "environment": {
        "allowlist": [
          "EXAMPLE_HOST"
        ]
      },
      "params": {
        "path": "{{.GetParam \"rest\"}}",
        "host": "{{.GetEnv \"EXAMPLE_HOST\"}}",
        "query": "{{.GetQuery \"query\"}}",
        "contentType": "{{.GetHeader \"content-type\"}}"
      },
      "checks": [
        {
          "expr": "host != \"\"",
          "error": "host is required"
        },
        {
          "expr": "path != \"\"",
          "error": "path is required"
        }
      ],
      "redirect": {
        "url": "https://{{.host}}/any/will/do/{{.path}}?q={{.query}}&h={{.contentType}}"
      }
    }
  ]
}`, certPath, keyPath)),
		},
		{
			description:   "missing config",
			configPath:    "/this/does/not/exist",
			expectedError: errors.New("failed to read configuration file at /this/does/not/exist: open /this/does/not/exist: no such file or directory"),
		},
		{
			description:   "empty config",
			configPath:    writeConfigYAML(t, ""),
			expectedError: errors.New("failed to parse configuration file: EOF"),
		},
		{
			description:   "unsupported config",
			configPath:    writeTempFile(t, "{}", "config.xyz"),
			expectedError: errors.New("unsupported configuration file extension: \".xyz\" (supported are .yaml, .yml and .json)"),
		},
		{
			description:   "invalid YAML config",
			configPath:    writeConfigYAML(t, "server: []\nroutes: []\n"),
			expectedError: errors.New("failed to parse configuration file: yaml: unmarshal errors:\n  line 1: cannot unmarshal !!seq into server.InputConfig"),
		},
		{
			description:   "invalid JSONconfig",
			configPath:    writeConfigJSON(t, "{"),
			expectedError: errors.New("failed to parse configuration file: unexpected EOF"),
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()

			_, _, err := config.ParseConfigFile(test.configPath)
			if test.expectedError == nil && err == nil {
				return
			}

			if test.expectedError == nil && err != nil {
				t.Fatalf(`expected no error but got "%v"`, err)
			} else if test.expectedError != nil && err == nil {
				t.Fatalf(`expected error "%v" but got nil`, test.expectedError)
			} else if err.Error() != test.expectedError.Error() {
				t.Fatalf(`expected error "%v" but got "%v"`, test.expectedError, err)
			}
		})
	}
}
