package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestServe(t *testing.T) {
	t.Setenv("EXAMPLE_HOST", "localhost")

	ctx, cancelCtx := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancelCtx)

	serverCtx, cancelServerCtx := context.WithCancel(ctx)
	t.Cleanup(cancelServerCtx)

	errCh := make(chan error, 1)

	go func() {
		defer close(errCh)

		os.Args = []string{"murl", "serve", "--config", filepath.Join("testdata", "config.yaml")}
		if result := run(serverCtx); result != 0 {
			errCh <- fmt.Errorf("unexpected exit code: %d", result)
		}
	}()

	client := http.Client{
		Timeout: 1 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: http.DefaultTransport,
	}

	ticker := time.NewTicker(10 * time.Millisecond)
	for {
		select {
		case err := <-errCh:
			if err != nil {
				t.Fatalf("server failed to start: %v", err)
			}

			return
		case <-ctx.Done():
			t.Fatalf("server did not start and respond with 307 TemporaryRedirect to test request in time")
		case <-ticker.C:
			resp, err := client.Get("http://localhost:8080/example/test")
			if err == nil && resp.StatusCode == http.StatusTemporaryRedirect {
				cancelServerCtx()
			} else if resp.StatusCode == http.StatusBadRequest {
				content, _ := io.ReadAll(resp.Body)
				t.Logf("expected 307 TemporaryRedirect but got %s / %v", resp.Status, string(content))
			}
		}
	}
}

func TestValidate(t *testing.T) {
	ctx, cancelCtx := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancelCtx)

	os.Args = []string{"murl", "validate", "--config", filepath.Join("testdata", "config.yaml")}
	if result := run(ctx); result != 0 {
		t.Fatalf("unexpected exit code: %d", result)
	}
}
