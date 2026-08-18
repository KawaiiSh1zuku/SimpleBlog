package main

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestNewHTTPServerHasTimeouts(t *testing.T) {
	server := newHTTPServer("127.0.0.1:0", http.NewServeMux())

	if server.ReadHeaderTimeout <= 0 || server.ReadTimeout <= 0 || server.WriteTimeout <= 0 || server.IdleTimeout <= 0 {
		t.Fatalf("server timeouts must all be positive: %+v", server)
	}
}

func TestServeStopsWhenContextIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	server := newHTTPServer("127.0.0.1:0", http.NewServeMux())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	if err := serve(ctx, server); err != nil {
		t.Fatalf("serve() returned an error during graceful shutdown: %v", err)
	}
}
