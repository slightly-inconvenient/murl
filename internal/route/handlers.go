package route

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"

	"github.com/google/cel-go/common/types"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 256))
	},
}

type paramsInput struct {
	getPath   func(key string) string
	getQuery  func(key string) string
	getHeader func(key string) string
	getEnv    func(key string) string
}

func (s *paramsInput) GetPath(key string) string {
	return s.getPath(key)
}

func (s *paramsInput) GetQuery(key string) string {
	return s.getQuery(key)
}

func (s *paramsInput) GetHeader(key string) string {
	return s.getHeader(key)
}

func (s *paramsInput) GetEnv(key string) string {
	return s.getEnv(key)
}

type Handler struct {
	path    string
	handler http.HandlerFunc
}

func (s Handler) Route() string {
	return "GET " + s.path
}

func (s Handler) Handler() http.HandlerFunc {
	return s.handler
}

func NewHandlers(routes []Route) []Handler {
	handlers := make([]Handler, 0, len(routes))
	for idx, route := range routes {
		if !route.valid {
			panic(fmt.Errorf("route at index %d has not been validated - create the routes using NewRoutes", idx))
		}
		handler := createRouteHandler(route)
		for _, path := range route.paths {
			handlers = append(handlers, Handler{path: path, handler: handler})
		}
	}
	return handlers
}

func TestHandlers(ctx context.Context, routes []Route, handlers []Handler) error {
	mux := http.NewServeMux()
	for _, handler := range handlers {
		mux.HandleFunc(handler.Route(), handler.Handler())
	}

	for idx, route := range routes {
		if !route.valid {
			panic(fmt.Errorf("route at index %d has not been validated - create the routes using NewRoutes", idx))
		}

		for _, test := range route.tests {
			if err := testRoute(ctx, mux, test); err != nil {
				return err
			}
		}
	}

	return nil
}

func createRouteHandler(route Route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		input := &paramsInput{
			getPath: func(key string) string {
				return r.PathValue(key)
			},
			getQuery: func(key string) string {
				return r.URL.Query().Get(key)
			},
			getHeader: func(key string) string {
				return r.Header.Get(key)
			},
			getEnv: func(key string) string {
				if route.environment.allowedEnvVariables[key] {
					return os.Getenv(key)
				}

				return ""
			},
		}

		params := map[string]any{}
		for key, expr := range route.params {
			buffer, release := getBuffer()
			defer release()

			err := expr.Execute(buffer, input)
			if err != nil {
				http.Error(w, fmt.Sprintf("failed to parse param for key %q: %s", key, err), http.StatusBadRequest)
				return
			}
			params[key] = buffer.String()
		}

		for _, check := range route.checks {
			out, _, err := check.expr.Eval(params)
			if err != nil {
				http.Error(w, fmt.Sprintf("failed to evaluate check expression: %s", err), http.StatusBadRequest)
				return
			}

			if out != types.True {
				buffer, release := getBuffer()
				defer release()
				if err := check.error.Execute(buffer, params); err != nil {
					http.Error(w, fmt.Sprintf("failed to render check error: %s", err), http.StatusBadRequest)
					return
				}

				http.Error(w, buffer.String(), http.StatusBadRequest)
				return
			}
		}

		buffer, release := getBuffer()
		defer release()
		if err := route.redirect.url.Execute(buffer, params); err != nil {
			http.Error(w, fmt.Sprintf("failed to create redirect url: %s", err), http.StatusBadRequest)
			return
		}

		redirect := buffer.String()
		http.Redirect(w, r, redirect, http.StatusTemporaryRedirect)
	}
}

func getBuffer() (*bytes.Buffer, func()) {
	buffer := bufferPool.Get().(*bytes.Buffer)
	buffer.Reset()

	return buffer, func() {
		bufferPool.Put(buffer)
	}
}

func testRoute(ctx context.Context, mux *http.ServeMux, test RouteTest) error {
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, test.request.url, nil)
	for k, v := range test.request.headers {
		req.Header.Add(k, v)
	}

	clear := setCheckEnvironment(test.request.environment)
	defer clear()

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != test.response.status {
		return fmt.Errorf("expected status %d but got %d", test.response.status, w.Code)
	}

	if w.Header().Get("Location") != test.response.url {
		return fmt.Errorf("expected redirect to %q but got %q", test.response.url, w.Header().Get("Location"))
	}

	return nil
}

func setCheckEnvironment(env map[string]string) func() {
	setEnv := func(key string, value string) {
		if err := os.Setenv(key, value); err != nil {
			panic(fmt.Errorf("failed to set env %w", err))
		}
	}
	unsetEnv := func(key string) {
		if err := os.Unsetenv(key); err != nil {
			panic(fmt.Errorf("failed to unset env %w", err))
		}
	}

	originalEnv := map[string]*string{}
	for key, newValue := range env {
		existing, existed := os.LookupEnv(key)
		if existed {
			originalEnv[key] = &existing
		} else {
			originalEnv[key] = nil
		}

		setEnv(key, newValue)
	}

	return func() {
		for k, v := range originalEnv {
			if v == nil {
				unsetEnv(k)
			} else {
				setEnv(k, *v)
			}
		}
	}
}
