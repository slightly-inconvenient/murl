package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

func Run(ctx context.Context, config Config, mux *http.ServeMux) error {
	if !config.valid {
		panic(errors.New("server config has not been validated - create the config using NewServerConfig"))
	}

	server := &http.Server{
		Addr:    config.address,
		Handler: mux,
	}

	closed := make(chan struct{})

	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
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
