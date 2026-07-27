package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/basefoundry/banyanlabs/services/url-shortener/internal/app"
	"github.com/basefoundry/banyanlabs/services/url-shortener/internal/logging"
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

func TestRequestLoggingUsesBanyanLabsContractFields(t *testing.T) {
	var logs bytes.Buffer
	logger := logging.New(&logs, "info", "url-shortener")
	application := app.New(app.Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	server := New(Options{
		Addr:   "127.0.0.1:0",
		App:    application,
		Logger: logger,
	})

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	request.Header.Set("X-Request-ID", "request-123")
	request.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	response := httptest.NewRecorder()

	server.Handler().ServeHTTP(response, request)

	logLines := strings.Split(strings.TrimSpace(logs.String()), "\n")
	if len(logLines) == 0 || logLines[0] == "" {
		t.Fatal("missing request log line")
	}

	var event map[string]any
	if err := json.Unmarshal([]byte(logLines[len(logLines)-1]), &event); err != nil {
		t.Fatalf("unmarshal request log event: %v; logs=%s", err, logs.String())
	}

	for _, key := range []string{
		"timestamp",
		"level",
		"message",
		"service",
		"component",
		"request_id",
		"trace_id",
		"span_id",
		"method",
		"path",
		"status",
		"duration_ms",
	} {
		if _, ok := event[key]; !ok {
			t.Fatalf("expected log key %q in %#v", key, event)
		}
	}

	expected := map[string]any{
		"level":      "info",
		"message":    "request completed",
		"service":    "url-shortener",
		"component":  "http",
		"request_id": "request-123",
		"trace_id":   "4bf92f3577b34da6a3ce929d0e0e4736",
		"span_id":    "00f067aa0ba902b7",
		"method":     http.MethodGet,
		"path":       "/healthz",
	}
	for key, want := range expected {
		if event[key] != want {
			t.Fatalf("%s = %v, want %v", key, event[key], want)
		}
	}
	if event["status"] != float64(http.StatusOK) {
		t.Fatalf("status = %v, want %d", event["status"], http.StatusOK)
	}
}
