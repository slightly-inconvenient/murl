package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/slightly-inconvenient/murl"
)

func Run(ctx context.Context, config Config, handlers []murl.Handler) error {
	if !config.valid {
		panic(errors.New("server config has not been validated - create the config using NewServerConfig"))
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET "+config.documentation.path, createDocsHandler(config.documentation.content))
	for _, handler := range handlers {
		mux.HandleFunc(handler.Path(), handler.Handler())
	}

	server := &http.Server{
		Addr:    config.address,
		Handler: mux,
	}

	closed := make(chan struct{})

	go func() {
		<-ctx.Done()
		_ = server.Shutdown(context.Background())
		close(closed)
	}()

	fmt.Println("Starting server on", config.address)

	result := error(nil)
	if config.tls.cert != "" && config.tls.key != "" {
		result = server.ListenAndServeTLS(config.tls.cert, config.tls.key)
	} else {
		result = server.ListenAndServe()
	}

	<-closed

	if errors.Is(result, http.ErrServerClosed) {
		return nil
	}

	return result
}

func createDocsHandler(documentation []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write(documentation)
	}
}
