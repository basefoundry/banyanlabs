package httpserver

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/basefoundry/banyanlabs/services/url-shortener/internal/app"
	"github.com/basefoundry/banyanlabs/services/url-shortener/internal/storage/sqlite"
)

func TestSignupEndpointCreatesSessionCookie(t *testing.T) {
	server, _ := newAuthTestServer(t)

	response := postJSON(t, server, "/auth/signup", `{
		"username": "alice",
		"email": "alice@example.com",
		"password": "correct horse battery staple"
	}`, nil)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusCreated, response.Body.String())
	}
	cookie := requireSessionCookie(t, response.Result())
	if !cookie.HttpOnly {
		t.Fatal("session cookie is not HttpOnly")
	}
	if cookie.Path != "/" {
		t.Fatalf("session cookie path = %q, want /", cookie.Path)
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("session cookie SameSite = %d, want %d", cookie.SameSite, http.SameSiteLaxMode)
	}

	var body authResponse
	decodeJSON(t, response, &body)
	if body.UserID == 0 {
		t.Fatal("user_id = 0")
	}
	if body.Username != "alice" {
		t.Fatalf("username = %q", body.Username)
	}
}

func TestLoginEndpointCreatesSessionCookie(t *testing.T) {
	server, _ := newAuthTestServer(t)
	signup := postJSON(t, server, "/auth/signup", `{
		"username": "alice",
		"email": "alice@example.com",
		"password": "correct horse battery staple"
	}`, nil)
	if signup.Code != http.StatusCreated {
		t.Fatalf("signup status = %d, want %d; body=%s", signup.Code, http.StatusCreated, signup.Body.String())
	}

	response := postJSON(t, server, "/auth/login", `{
		"username": "alice",
		"password": "correct horse battery staple"
	}`, nil)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusOK, response.Body.String())
	}
	requireSessionCookie(t, response.Result())
	var body authResponse
	decodeJSON(t, response, &body)
	if body.Username != "alice" {
		t.Fatalf("username = %q", body.Username)
	}
}

func TestLoginEndpointRejectsInvalidCredentials(t *testing.T) {
	server, _ := newAuthTestServer(t)
	signup := postJSON(t, server, "/auth/signup", `{
		"username": "alice",
		"email": "alice@example.com",
		"password": "correct horse battery staple"
	}`, nil)
	if signup.Code != http.StatusCreated {
		t.Fatalf("signup status = %d, want %d; body=%s", signup.Code, http.StatusCreated, signup.Body.String())
	}

	response := postJSON(t, server, "/auth/login", `{
		"username": "alice",
		"password": "wrong password"
	}`, nil)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusUnauthorized, response.Body.String())
	}
	if strings.Contains(response.Header().Get("Set-Cookie"), sessionCookieName) {
		t.Fatal("invalid login set a session cookie")
	}
}

func TestLoginEndpointReturnsTooManyRequestsAfterRepeatedFailures(t *testing.T) {
	const lockoutAttempts = 5

	server, _ := newAuthTestServer(t)

	signup := postJSON(t, server, "/auth/signup", `{
		"username": "alice",
		"email": "alice@example.com",
		"password": "correct horse battery staple"
	}`, nil)
	if signup.Code != http.StatusCreated {
		t.Fatalf("signup status = %d, want %d; body=%s", signup.Code, http.StatusCreated, signup.Body.String())
	}

	for attempt := 0; attempt < lockoutAttempts; attempt++ {
		response := postJSON(t, server, "/auth/login", `{
			"username": "alice",
			"password": "wrong password"
		}`, nil)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d status = %d, want %d; body=%s", attempt+1, response.Code, http.StatusUnauthorized, response.Body.String())
		}
	}

	response := postJSON(t, server, "/auth/login", `{
		"username": "alice",
		"password": "correct horse battery staple"
	}`, nil)
	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("locked status = %d, want %d; body=%s", response.Code, http.StatusTooManyRequests, response.Body.String())
	}
}

func TestLogoutEndpointClearsSessionCookie(t *testing.T) {
	server, _ := newAuthTestServer(t)
	signup := postJSON(t, server, "/auth/signup", `{
		"username": "alice",
		"email": "alice@example.com",
		"password": "correct horse battery staple"
	}`, nil)
	if signup.Code != http.StatusCreated {
		t.Fatalf("signup status = %d, want %d; body=%s", signup.Code, http.StatusCreated, signup.Body.String())
	}

	response := postJSON(t, server, "/auth/logout", `{}`, requireSessionCookie(t, signup.Result()))

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusNoContent, response.Body.String())
	}
	setCookie := response.Header().Get("Set-Cookie")
	if !strings.Contains(setCookie, sessionCookieName+"=") {
		t.Fatalf("missing cleared session cookie: %q", setCookie)
	}
	if !strings.Contains(setCookie, "Max-Age=0") {
		t.Fatalf("session cookie was not expired: %q", setCookie)
	}
}

func TestRequireUserAllowsValidSession(t *testing.T) {
	server, _ := newAuthTestServer(t)
	signup := postJSON(t, server, "/auth/signup", `{
		"username": "alice",
		"email": "alice@example.com",
		"password": "correct horse battery staple"
	}`, nil)
	if signup.Code != http.StatusCreated {
		t.Fatalf("signup status = %d, want %d; body=%s", signup.Code, http.StatusCreated, signup.Body.String())
	}

	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.AddCookie(requireSessionCookie(t, signup.Result()))
	response := httptest.NewRecorder()

	server.requireUser(func(writer http.ResponseWriter, _ *http.Request, user app.User) {
		_, _ = writer.Write([]byte(user.Username))
	}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusOK, response.Body.String())
	}
	if response.Body.String() != "alice" {
		t.Fatalf("body = %q", response.Body.String())
	}
}

func TestRequireUserRejectsMissingSession(t *testing.T) {
	server, _ := newAuthTestServer(t)
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	response := httptest.NewRecorder()

	server.requireUser(func(writer http.ResponseWriter, _ *http.Request, user app.User) {
		t.Fatalf("handler called with user %#v", user)
	}).ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusUnauthorized, response.Body.String())
	}
}

type authResponse struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
}

func newAuthTestServer(t *testing.T) (*Server, *sql.DB) {
	t.Helper()

	ctx := context.Background()
	db, err := sqlite.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("close database: %v", err)
		}
	})

	if err := sqlite.RunMigrations(ctx, db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	application := app.New(app.Options{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Store:  sqlite.NewStore(db),
	})
	server := New(Options{
		Addr:   "127.0.0.1:0",
		App:    application,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	return server, db
}

func postJSON(t *testing.T, server *Server, path string, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()

	server.Handler().ServeHTTP(response, request)
	return response
}

func requireSessionCookie(t *testing.T, response *http.Response) *http.Cookie {
	t.Helper()

	for _, cookie := range response.Cookies() {
		if cookie.Name == sessionCookieName {
			if cookie.Value == "" {
				t.Fatal("empty session cookie")
			}
			return cookie
		}
	}
	t.Fatalf("missing %s cookie in %v", sessionCookieName, response.Cookies())
	return nil
}

func decodeJSON(t *testing.T, response *httptest.ResponseRecorder, target any) {
	t.Helper()

	if err := json.Unmarshal(response.Body.Bytes(), target); err != nil {
		t.Fatalf("decode JSON response: %v; body=%s", err, response.Body.String())
	}
}
