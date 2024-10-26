package murl

import (
	"bytes"
	"fmt"
	"net/http"
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

func NewMux(routes []Route) *http.ServeMux {
	mux := http.NewServeMux()
	for idx, route := range routes {
		if !route.valid {
			panic(fmt.Errorf("route at index %d has not been validated - create the routes using NewRoutes", idx))
		}
		handler := createRouteHandler(route)
		for _, path := range route.paths {
			mux.HandleFunc("GET "+path, handler)
		}
	}
	return mux
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
