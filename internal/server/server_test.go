package server_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/slightly-inconvenient/murl/internal/server"
	"github.com/slightly-inconvenient/murl/internal/testtls"
)

func TestRun(t *testing.T) {
	t.Parallel()

	t.Run("panics if config has not been validated", func(t *testing.T) {
		t.Parallel()
		defer func() {
			r := recover()
			if r == nil {
				t.Fatalf("expected panic but did not get one")
			}
			err, ok := r.(error)
			if !ok {
				t.Fatalf("expected panic to be of type error but got %T", r)
			}

			expectedErrorMsg := "server config has not been validated - create the config using NewServerConfig"
			if err.Error() != expectedErrorMsg {
				t.Fatalf("expected panic to be '%s' but got %v", expectedErrorMsg, err)
			}
		}()

		if err := server.Run(context.Background(), server.Config{}, nil); err != nil {
			t.Fatalf("expected error to be nil but got %v", err)
		}
	})

	caFile, certFile, keyFile := testtls.CreateTestTLSCertificates(t.TempDir())
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: func() *x509.CertPool {
					caPool := x509.NewCertPool()
					caBytes, _ := os.ReadFile(caFile)
					if !caPool.AppendCertsFromPEM(caBytes) {
						panic(errors.New("failed to append client CA certificate"))
					}
					return caPool
				}(),
			},
		},
	}

	tests := []struct {
		description string
		config      server.InputConfig
	}{
		{
			description: "starts without TLS",
			config: server.InputConfig{
				Address: "localhost:8080",
			},
		},
		{
			description: "starts with TLS",
			config: server.InputConfig{
				Address: "localhost:8443",
				TLS: server.InputTLSConfig{
					Cert: certFile,
					Key:  keyFile,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()

			ctx, cancelCtx := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancelCtx()

			path := "/test"
			mux := http.NewServeMux()
			mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			errCh := make(chan error)
			config, err := server.NewConfig(test.config)
			if err != nil {
				t.Fatalf("failed to create test server config: %v", err)
			}

			go func() {
				errCh <- server.Run(ctx, config, mux)
				close(errCh)
			}()

			func() {
				for {
					select {
					case <-ctx.Done():
						t.Fatalf("server did not start and respond with 200 OK to test request in time")
					default:

						url := "http://" + test.config.Address + path
						if test.config.TLS.Cert != "" && test.config.TLS.Key != "" {
							url = strings.Replace(url, "http:", "https:", 1)
						}

						resp, err := httpClient.Get(url)
						if err == nil && resp.StatusCode == http.StatusOK {
							cancelCtx()
							return
						}

						time.Sleep(50 * time.Millisecond)
					}
				}
			}()

			if err := <-errCh; err != nil {
				t.Fatalf("server failed to start: %v", err)
			}
		})
	}
}
