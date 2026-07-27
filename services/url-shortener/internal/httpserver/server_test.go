package httpserver

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/basefoundry/banyanlabs/services/url-shortener/internal/app"
)

func TestHealthEndpoint(t *testing.T) {
	application := app.New(app.Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	server := New(Options{
		Addr:   "127.0.0.1:0",
		App:    application,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()

	server.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Header().Get("X-Request-ID") == "" {
		t.Fatal("missing X-Request-ID response header")
	}

	var health app.Health
	if err := json.Unmarshal(response.Body.Bytes(), &health); err != nil {
		t.Fatalf("unmarshal health response: %v", err)
	}
	if health.Service != "url-shortener" {
		t.Fatalf("service = %q", health.Service)
	}
	if health.Status != "ok" {
		t.Fatalf("status = %q", health.Status)
	}
}

func TestHealthEndpointRejectsUnsupportedMethods(t *testing.T) {
	application := app.New(app.Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	server := New(Options{
		Addr:   "127.0.0.1:0",
		App:    application,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	request := httptest.NewRequest(http.MethodPost, "/healthz", nil)
	response := httptest.NewRecorder()

	server.Handler().ServeHTTP(response, request.WithContext(context.Background()))

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
}
