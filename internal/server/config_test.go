package server_test

import (
	"errors"
	"testing"

	"github.com/slightly-inconvenient/murl"
	"github.com/slightly-inconvenient/murl/internal/server"
	"github.com/slightly-inconvenient/murl/internal/testtls"
)

type serverConfigTestTransform func(*server.InputConfig)

func buildTestServerConfig(transforms ...serverConfigTestTransform) server.InputConfig {
	config := server.InputConfig{
		Address: ":8080",
	}

	for _, transform := range transforms {
		transform(&config)
	}

	return config
}

func TestConfig_Succes(t *testing.T) {
	t.Parallel()

	t.Run("http server", func(t *testing.T) {
		t.Parallel()
		input := buildTestServerConfig()
		_, err := server.NewConfig(input, []murl.InputRoute{})
		if err != nil {
			t.Fatalf("expected create server config to succeed but got error: %s", err)
		}
	})

	t.Run("https server", func(t *testing.T) {
		t.Parallel()
		_, certFile, keyFile := testtls.CreateTestTLSCertificates(t.TempDir())
		input := buildTestServerConfig(func(ic *server.InputConfig) {
			ic.TLS = server.InputTLSConfig{
				Cert: certFile,
				Key:  keyFile,
			}
		})
		_, err := server.NewConfig(input, []murl.InputRoute{})
		if err != nil {
			t.Fatalf("expected create server config to succeed but got error: %s", err)
		}
	})
}

func TestConfig_Failures(t *testing.T) {
	t.Parallel()
	_, certFile, keyFile := testtls.CreateTestTLSCertificates(t.TempDir())

	tests := []struct {
		description   string
		config        server.InputConfig
		routes        []murl.InputRoute
		expectedError error
	}{
		{
			description: "fails with server address missing",
			config: buildTestServerConfig(func(ic *server.InputConfig) {
				ic.Address = ""
			}),
			routes:        []murl.InputRoute{},
			expectedError: errors.New("server address is required"),
		},
		{
			description: "fails with tls key missing when cert provided",
			config: buildTestServerConfig(func(ic *server.InputConfig) {
				ic.TLS = server.InputTLSConfig{
					Cert: "/path/does/not/exist",
				}
			}),
			routes:        []murl.InputRoute{},
			expectedError: errors.New("server TLS key is required when TLS cert is provided"),
		},
		{
			description: "fails with tls cert missing when key provided",
			config: buildTestServerConfig(func(ic *server.InputConfig) {
				ic.TLS = server.InputTLSConfig{
					Key: "/path/does/not/exist",
				}
			}),
			routes:        []murl.InputRoute{},
			expectedError: errors.New("server TLS cert is required when TLS key is provided"),
		},
		{
			description: "fails with invalid tls key when cert provided",
			config: buildTestServerConfig(func(ic *server.InputConfig) {
				ic.TLS = server.InputTLSConfig{
					Cert: certFile,
					Key:  "/path/does/not/exist",
				}
			}),
			routes:        []murl.InputRoute{},
			expectedError: errors.New("server TLS key file at path \"/path/does/not/exist\" does not exist"),
		},
		{
			description: "fails with invalid tls cert when key provided",
			config: buildTestServerConfig(func(ic *server.InputConfig) {
				ic.TLS = server.InputTLSConfig{
					Cert: "/path/does/not/exist",
					Key:  keyFile,
				}
			}),
			routes:        []murl.InputRoute{},
			expectedError: errors.New("server TLS cert file at path \"/path/does/not/exist\" does not exist"),
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()

			_, err := server.NewConfig(test.config, test.routes)
			if err == nil {
				t.Fatalf("expected create server config to fail but got nil")
			}
			if err.Error() != test.expectedError.Error() {
				t.Fatalf("expected error to be %s but got %s", test.expectedError, err)
			}
		})
	}
}
