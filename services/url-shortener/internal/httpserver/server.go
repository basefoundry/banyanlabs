package httpserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/basefoundry/banyanlabs/services/url-shortener/internal/app"
)

type App interface {
	Health(context.Context) app.Health
	Signup(context.Context, app.SignupInput) (app.AuthResult, error)
	Login(context.Context, app.LoginInput) (app.AuthResult, error)
	Logout(context.Context, string) error
	CurrentUser(context.Context, string) (app.User, error)
}

type Options struct {
	Addr   string
	App    App
	Logger *slog.Logger
}

type Server struct {
	app    App
	logger *slog.Logger
	server *http.Server
}

func New(options Options) *Server {
	logger := options.Logger
	if logger == nil {
		logger = slog.Default()
	}
	server := &Server{
		app:    options.App,
		logger: logger.With(slog.String("component", "http")),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", server.handleHealth)
	mux.HandleFunc("/auth/signup", server.handleSignup)
	mux.HandleFunc("/auth/login", server.handleLogin)
	mux.HandleFunc("/auth/logout", server.handleLogout)

	server.server = &http.Server{
		Addr:              options.Addr,
		Handler:           server.logRequests(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}
	return server
}

func (server *Server) Handler() http.Handler {
	return server.server.Handler
}

func (server *Server) ListenAndServe(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		err := server.server.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		errCh <- err
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return <-errCh
	case err := <-errCh:
		return err
	}
}

func (server *Server) handleHealth(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		writer.Header().Set("Allow", "GET, HEAD")
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	if request.Method == http.MethodHead {
		return
	}
	if err := json.NewEncoder(writer).Encode(server.app.Health(request.Context())); err != nil {
		server.logger.Error("failed to encode health response", slog.Any("error", err))
	}
}

func (server *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		start := time.Now()
		requestID := request.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = newRequestID()
		}

		recorder := &statusRecorder{ResponseWriter: writer, status: http.StatusOK}
		recorder.Header().Set("X-Request-ID", requestID)

		next.ServeHTTP(recorder, request)

		attrs := []any{
			slog.String("request_id", requestID),
			slog.String("method", request.Method),
			slog.String("path", request.URL.Path),
			slog.Int("status", recorder.status),
			slog.Int64("duration_ms", time.Since(start).Milliseconds()),
		}
		if traceID, spanID := requestTraceContext(request); traceID != "" {
			attrs = append(attrs, slog.String("trace_id", traceID))
			if spanID != "" {
				attrs = append(attrs, slog.String("span_id", spanID))
			}
		}

		server.logger.Info("request completed", attrs...)
	})
}

func requestTraceContext(request *http.Request) (string, string) {
	if traceparent := request.Header.Get("traceparent"); traceparent != "" {
		parts := strings.Split(traceparent, "-")
		if len(parts) >= 4 && validTraceID(parts[1]) && validSpanID(parts[2]) {
			return parts[1], parts[2]
		}
	}

	traceID := strings.TrimSpace(request.Header.Get("X-Trace-ID"))
	spanID := strings.TrimSpace(request.Header.Get("X-Span-ID"))
	return traceID, spanID
}

func validTraceID(traceID string) bool {
	return len(traceID) == 32 && traceID != "00000000000000000000000000000000" && isLowerHex(traceID)
}

func validSpanID(spanID string) bool {
	return len(spanID) == 16 && spanID != "0000000000000000" && isLowerHex(spanID)
}

func isLowerHex(value string) bool {
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (recorder *statusRecorder) WriteHeader(status int) {
	recorder.status = status
	recorder.ResponseWriter.WriteHeader(status)
}

func newRequestID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(bytes[:])
}
