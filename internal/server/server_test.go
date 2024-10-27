package server_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/slightly-inconvenient/murl/internal/config"
	"github.com/slightly-inconvenient/murl/internal/route"
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
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
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

	checkDocs := func(expectedContent string) func(t *testing.T, resp *http.Response) {
		return func(t *testing.T, resp *http.Response) {
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected status code to be 200 OK but got %s", resp.Status)
			}

			contentType := resp.Header.Get("Content-Type")
			if !strings.Contains(contentType, "text/html") {
				t.Fatalf("expected content type to be text/html but got %s", contentType)
			}

			content, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read response body: %v", err)
			}

			if string(content) != expectedContent {
				t.Fatalf("expected content to be %q but got %q", expectedContent, string(content))
			}
		}
	}

	tests := []struct {
		description string
		config      config.Server
		routes      []config.Route
		requestPath string
		check       func(t *testing.T, resp *http.Response)
	}{
		{
			description: "serves routes without TLS",
			config: config.Server{
				Address: "localhost:8080",
			},
			routes: []config.Route{
				{Path: "/test", Redirect: config.RouteRedirect{URL: "http://localhost:8080/test2"}},
			},
			requestPath: "/test",
			check: func(t *testing.T, resp *http.Response) {
				if resp.StatusCode != http.StatusTemporaryRedirect {
					t.Fatalf("expected status code to be 307 Temporary Redirect but got %s", resp.Status)
				}
			},
		},
		{
			description: "serves routes with TLS",
			config: config.Server{
				Address: "localhost:8443",
				TLS: config.ServerTLSConfig{
					Cert: certFile,
					Key:  keyFile,
				},
			},
			routes: []config.Route{
				{Path: "/test", Redirect: config.RouteRedirect{URL: "http://localhost:8080/test2"}},
			},
			requestPath: "/test",
			check: func(t *testing.T, resp *http.Response) {
				if resp.StatusCode != http.StatusTemporaryRedirect {
					t.Fatalf("expected status code to be 307 Temporary Redirect but got %s", resp.Status)
				}
			},
		},
		{
			description: "serves docs from default path",
			config: config.Server{
				Address: "localhost:8081",
			},
			routes:      []config.Route{},
			requestPath: "",
			check:       checkDocs("<h1 id=\"available-routes\">Available Routes</h1>\n"),
		},
		{
			description: "serves docs from custom path",
			config: config.Server{
				Address: "localhost:8082",
				Documentation: config.ServerDocumentationConfig{
					Path: "/docs",
				},
			},
			routes:      []config.Route{},
			requestPath: "/docs",
			check:       checkDocs("<h1 id=\"available-routes\">Available Routes</h1>\n"),
		},
		{
			description: "serves custom docs template",
			config: config.Server{
				Address: "localhost:8083",
				Documentation: config.ServerDocumentationConfig{
					Templates: config.ServerTemplatesConfig{
						Root: `
# Title
test custom template with {{ range . }} {{ .Path }} {{ end }}

including default routes template

{{ template "routes" . }}
`,
					},
				},
			},
			routes: []config.Route{
				{Path: "/test", Redirect: config.RouteRedirect{URL: "http://localhost:8080/test2"}},
			},
			requestPath: "",
			check:       checkDocs("<h1 id=\"title\">Title</h1>\n<p>test custom template with  /test</p>\n<p>including default routes template</p>\n<h1 id=\"available-routes\">Available Routes</h1>\n"),
		},
		{
			description: "serves route docs",
			config: config.Server{
				Address: "localhost:8084",
			},
			routes: []config.Route{
				{
					Path:    "/test",
					Aliases: []string{"/test-alias"},
					Documentation: config.RouteDocumentation{
						Title:       "Test Route",
						Description: "A test route",
					},
					Redirect: config.RouteRedirect{URL: "http://localhost:8080/test2"},
				},
			},
			requestPath: "",
			check:       checkDocs("<h1 id=\"available-routes\">Available Routes</h1>\n<h2 id=\"test-route\">Test Route</h2>\n<p>A test route</p>\n"),
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()

			ctx, cancelCtx := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancelCtx()

			routes, err := route.NewRoutes(test.routes)
			if err != nil {
				t.Fatalf("expected create routes to succeed but got error: %s", err)
			}

			errCh := make(chan error)
			config, err := server.NewConfig(test.config, test.routes)
			if err != nil {
				t.Fatalf("failed to create test server config: %v", err)
			}

			go func() {
				errCh <- server.Run(ctx, config, route.NewHandlers(routes))
				close(errCh)
			}()

			func() {
				for {
					select {
					case <-ctx.Done():
						t.Fatalf("server did not start and respond with 200 OK to test request in time")
					default:

						baseURL := "http://" + test.config.Address
						if test.config.TLS.Cert != "" && test.config.TLS.Key != "" {
							baseURL = strings.Replace(baseURL, "http:", "https:", 1)
						}

						resp, err := httpClient.Get(baseURL + test.requestPath)
						if err == nil {
							test.check(t, resp)
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
