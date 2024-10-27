package server_test

import (
	"errors"
	"testing"

	"github.com/slightly-inconvenient/murl/internal/config"
	"github.com/slightly-inconvenient/murl/internal/server"
	"github.com/slightly-inconvenient/murl/internal/testtls"
)

type serverConfigTestTransform func(*config.Server)

func buildTestServerConfig(transforms ...serverConfigTestTransform) config.Server {
	config := config.Server{
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
		_, err := server.NewConfig(input, []config.Route{})
		if err != nil {
			t.Fatalf("expected create server config to succeed but got error: %s", err)
		}
	})

	t.Run("https server", func(t *testing.T) {
		t.Parallel()
		_, certFile, keyFile := testtls.CreateTestTLSCertificates(t.TempDir())
		input := buildTestServerConfig(func(ic *config.Server) {
			ic.TLS = config.ServerTLSConfig{
				Cert: certFile,
				Key:  keyFile,
			}
		})
		_, err := server.NewConfig(input, []config.Route{})
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
		config        config.Server
		routes        []config.Route
		expectedError error
	}{
		{
			description: "fails with server address missing",
			config: buildTestServerConfig(func(ic *config.Server) {
				ic.Address = ""
			}),
			routes:        []config.Route{},
			expectedError: errors.New("server address is required"),
		},
		{
			description: "fails with tls key missing when cert provided",
			config: buildTestServerConfig(func(ic *config.Server) {
				ic.TLS = config.ServerTLSConfig{
					Cert: "/path/does/not/exist",
				}
			}),
			routes:        []config.Route{},
			expectedError: errors.New("server TLS key is required when TLS cert is provided"),
		},
		{
			description: "fails with tls cert missing when key provided",
			config: buildTestServerConfig(func(ic *config.Server) {
				ic.TLS = config.ServerTLSConfig{
					Key: "/path/does/not/exist",
				}
			}),
			routes:        []config.Route{},
			expectedError: errors.New("server TLS cert is required when TLS key is provided"),
		},
		{
			description: "fails with invalid tls key when cert provided",
			config: buildTestServerConfig(func(ic *config.Server) {
				ic.TLS = config.ServerTLSConfig{
					Cert: certFile,
					Key:  "/path/does/not/exist",
				}
			}),
			routes:        []config.Route{},
			expectedError: errors.New("server TLS key file at path \"/path/does/not/exist\" does not exist"),
		},
		{
			description: "fails with invalid tls cert when key provided",
			config: buildTestServerConfig(func(ic *config.Server) {
				ic.TLS = config.ServerTLSConfig{
					Cert: "/path/does/not/exist",
					Key:  keyFile,
				}
			}),
			routes:        []config.Route{},
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
