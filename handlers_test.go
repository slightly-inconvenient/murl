package murl_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/slightly-inconvenient/murl"
)

func createResponseChecker(expectedStatusCode int, expectedLocationOrBody string) func(*httptest.ResponseRecorder) error {
	return func(rec *httptest.ResponseRecorder) error {
		if rec.Code != expectedStatusCode {
			return fmt.Errorf("expected status code %d but got %d (body: %s)", expectedStatusCode, rec.Code, rec.Body.String())
		}

		if expectedStatusCode == http.StatusTemporaryRedirect {
			if rec.Header().Get("Location") != expectedLocationOrBody {
				return fmt.Errorf("expected location %s but got %s", expectedLocationOrBody, rec.Header().Get("Location"))
			}
			return nil
		}

		body := strings.Trim(rec.Body.String(), "\n")
		if body != expectedLocationOrBody {
			return fmt.Errorf("expected body %q but got %q", expectedLocationOrBody, body)
		}

		return nil
	}
}

func TestHandler(t *testing.T) {
	t.Parallel()

	t.Run("panics if a route has not been validated", func(t *testing.T) {
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

			expectedErrorMsg := "route at index 0 has not been validated - create the routes using NewRoutes"
			if err.Error() != expectedErrorMsg {
				t.Fatalf("expected panic to be %q but got %q", expectedErrorMsg, err)
			}
		}()

		murl.NewMux([]murl.Route{{}})
	})

	tests := []struct {
		description   string
		routes        []murl.InputRoute
		req           *http.Request
		checkResponse func(*httptest.ResponseRecorder) error
	}{
		{
			description: "simple route with no params",
			routes: []murl.InputRoute{
				{
					Path: "/example",
					Redirect: murl.InputRouteRedirect{
						URL: "https://example.com",
					},
				},
			},
			req:           httptest.NewRequest("GET", "/example", nil),
			checkResponse: createResponseChecker(http.StatusTemporaryRedirect, "https://example.com"),
		},
		{
			description: "complex params route",
			routes: []murl.InputRoute{
				{
					Path: "/example/{id}",
					Environment: murl.InputRouteEnvironment{
						Allowlist: []string{"TEST_KEY_HOST"},
					},
					Redirect: murl.InputRouteRedirect{
						URL: "https://{{.host}}/id/{{.id}}?query={{.q}}&header={{.h}}&envBlocked=\"{{.envBlocked}}\"",
					},
					Params: map[string]string{
						"id":         `{{.GetPath "id"}}`,
						"q":          `{{.GetQuery "q"}}`,
						"h":          `{{.GetHeader "x-test-header"}}`,
						"host":       `{{.GetEnv "TEST_KEY_HOST"}}`,
						"envBlocked": `{{.GetEnv "TEST_KEY_HOST_BLOCKED"}}`,
					},
					Checks: []murl.InputRouteCheck{
						{
							Expr:  `q != ""`,
							Error: "query variable q is required",
						},
						{
							Expr:  `host != ""`,
							Error: "host failed to parse from environment - unable to redirect",
						},
					},
				},
			},
			req: func() *http.Request {
				os.Setenv("TEST_KEY_HOST", "example.com")
				req := httptest.NewRequest("GET", "/example/wasd?q=xyz", nil)
				req.Header.Set("x-test-header", "abc")
				return req
			}(),
			checkResponse: createResponseChecker(http.StatusTemporaryRedirect, "https://example.com/id/wasd?query=xyz&header=abc&envBlocked=\"\""),
		},
		{
			description: "route with failing checks",
			routes: []murl.InputRoute{
				{
					Path: "/example",
					Redirect: murl.InputRouteRedirect{
						URL: "https://example.com/id/{{.id}}",
					},
					Params: map[string]string{
						"id": `{{.GetQuery "id"}}`,
					},
					Checks: []murl.InputRouteCheck{
						{
							Expr:  `id != ""`,
							Error: "id is required",
						},
					},
				},
			},
			req:           httptest.NewRequest("GET", "/example", nil),
			checkResponse: createResponseChecker(http.StatusBadRequest, "id is required"),
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()

			routes, err := murl.NewRoutes(test.routes)
			if err != nil {
				t.Fatalf("failed to create test routes: %v", err)
			}

			mux := murl.NewMux(routes)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, test.req)

			if err := test.checkResponse(rec); err != nil {
				t.Fatalf("unexpected response: %s", err)
			}
		})
	}
}
