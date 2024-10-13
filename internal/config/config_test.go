package config_test

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/slightly-inconvenient/murl/internal/config"
	"github.com/slightly-inconvenient/murl/internal/testtls"
)

func writeConfig(t *testing.T, config []byte) string {
	tempFile := t.TempDir() + "/config.yaml"
	err := os.WriteFile(tempFile, config, 0o644)
	if err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}

	return tempFile
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
			configPath: writeConfig(t, []byte(fmt.Sprintf(`
server:
  address: ":8443"
  tls:
    cert: %s
    key: %s
routes:
- path: /example/{rest}
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
 `, certPath, keyPath))),
		},
		{
			description: "valid JSON config",
			configPath: writeConfig(t, []byte(fmt.Sprintf(`{
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
      "description": "An example route that picks params from the env, path, query, and headers",
      "environment": {
        "allowlist": [
          "EXAMPLE_HOST"
        ]
      },
      "params": {
        "path": "{{.GetParam \"rest\"}}",
        "host": "{{.GetEnv \"EXAMPLE_HOST\"}}",
        "query": "{{.GetQuery \"query\"}}",
        "contentType": "{{.GetHeader \"content-type\"}}",
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
}`, certPath, keyPath))),
		},
		{
			description:   "missing config",
			configPath:    "/this/does/not/exist",
			expectedError: errors.New("failed to read configuration file at /this/does/not/exist: open /this/does/not/exist: no such file or directory"),
		},
		{
			description:   "empty config",
			configPath:    writeConfig(t, []byte("")),
			expectedError: errors.New("failed to parse configuration file: EOF"),
		},
		{
			description:   "invalid config",
			configPath:    writeConfig(t, []byte("server: []\nroutes: []\n")),
			expectedError: errors.New("failed to parse configuration file: yaml: unmarshal errors:\n  line 1: cannot unmarshal !!seq into server.InputConfig"),
		},
		{
			description: "invalid config with server address missing",
			configPath: writeConfig(t, []byte(`
server: {}
routes:
- path: /example
 `)),
			expectedError: errors.New("server address is required"),
		},
		{
			description: "invalid config with tls key missing when cert provided",
			configPath: writeConfig(t, []byte(fmt.Sprintf(`
server:
  address: ":8080"
  tls:
    cert: %s
routes:
- path: /example
 `, certPath))),
			expectedError: errors.New("server TLS key is required when TLS cert is provided"),
		},
		{
			description: "invalid config with tls cert missing when key provided",
			configPath: writeConfig(t, []byte(fmt.Sprintf(`
server:
  address: ":8080"
  tls:
    key: %s
routes:
- path: /example
 `, keyPath))),
			expectedError: errors.New("server TLS cert is required when TLS key is provided"),
		},
		{
			description: "invalid config with tls key missing when cert provided",
			configPath: writeConfig(t, []byte(fmt.Sprintf(`
server:
  address: ":8080"
  tls:
    cert: %s
    key: /path/does/not/exist
routes:
- path: /example
 `, certPath))),
			expectedError: errors.New("server TLS key file at path \"/path/does/not/exist\" does not exist"),
		},
		{
			description: "invalid config with tls cert missing when key provided",
			configPath: writeConfig(t, []byte(fmt.Sprintf(`
server:
  address: ":8080"
  tls:
    cert: /path/does/not/exist
    key: %s
routes:
- path: /example
 `, keyPath))),
			expectedError: errors.New("server TLS cert file at path \"/path/does/not/exist\" does not exist"),
		},
		{
			description: "invalid config with missing path",

			configPath: writeConfig(t, []byte(`
server:
  address: ":8080"
routes:
- {}
 `)),
			expectedError: errors.New("path to match against is required for route but missing from route at index 0"),
		},
		{
			description: "invalid config with non-absolute path",

			configPath: writeConfig(t, []byte(`
server:
  address: ":8080"
routes:
- path: example
 `)),
			expectedError: errors.New("path \"example\" must be an absolute path (start with slash)"),
		},
		{
			description: "invalid config with bad cel expr",
			configPath: writeConfig(t, []byte(`
server:
  address: ":8080"
routes:
- path: /example
  checks:
  - expr: '{'
    error: "bad expr"
 `)),
			expectedError: errors.New("failed to parse check expression for route /example: ERROR: <input>:1:2: Syntax error: mismatched input '<EOF>' expecting {'[', '{', '}', '(', '.', ',', '-', '!', '?', 'true', 'false', 'null', NUM_FLOAT, NUM_INT, NUM_UINT, STRING, BYTES, IDENTIFIER}\n | {\n | .^"),
		},
		{
			description: "invalid config with params input",
			configPath: writeConfig(t, []byte(`
server:
  address: ":8080"
routes:
- path: /example
  params:
    id: '{{}}}'
`)),
			expectedError: errors.New("failed to parse params for route /example: failed to parse param template \"id\": template: :1: missing value for command"),
		},
		{
			description: "invalid config with missing cel expr",
			configPath: writeConfig(t, []byte(`
server:
  address: ":8080"
routes:
- path: /example
  checks:
  - error: "bad expr"
 `)),
			expectedError: errors.New("failed to parse check expression for route /example: no expression to evaluate"),
		},
		{
			description: "invalid config with missing cel error",
			configPath: writeConfig(t, []byte(`
server:
  address: ":8080"
routes:
- path: /example
  checks:
  - expr: "true"
 `)),
			expectedError: errors.New("failed to parse check error template for route /example: missing template"),
		},
		{
			description: "invalid config with invalid cel error template",
			configPath: writeConfig(t, []byte(`
server:
  address: ":8080"
routes:
- path: /example
  checks:
  - expr: "true"
    error: "{{}}}"
 `)),
			expectedError: errors.New("failed to parse check error template for route /example: template: :1: missing value for command"),
		},
		{
			description: "invalid config with bad cel expr",
			configPath: writeConfig(t, []byte(`
server:
  address: ":8080"
routes:
- path: /example
  redirect:
    url: '{{}'
 `)),
			expectedError: errors.New("failed to parse redirect url for route /example: template: :1: unexpected \"}\" in command"),
		},
		{
			description: "invalid config with missing redirect url template",
			configPath: writeConfig(t, []byte(`
server:
  address: ":8080"
routes:
- path: /example
  redirect: {}
 `)),
			expectedError: errors.New("failed to parse redirect url for route /example: missing template"),
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
