package httpserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
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

		server.logger.Info(
			"request completed",
			slog.String("request_id", requestID),
			slog.String("method", request.Method),
			slog.String("path", request.URL.Path),
			slog.Int("status", recorder.status),
			slog.Int64("duration_ms", time.Since(start).Milliseconds()),
		)
	})
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
