package httpserver

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/basefoundry/banyanlabs/services/url-shortener/internal/app"
)

const sessionCookieName = "banyan_url_shortener_session"

type signupRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authResponseBody struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
}

type authenticatedHandler func(http.ResponseWriter, *http.Request, app.User)

func (server *Server) handleSignup(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writer.Header().Set("Allow", http.MethodPost)
		writeJSONError(writer, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var payload signupRequest
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		writeJSONError(writer, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := server.app.Signup(request.Context(), app.SignupInput{
		Username: payload.Username,
		Email:    payload.Email,
		Password: payload.Password,
	})
	if err != nil {
		server.writeAppError(writer, "signup failed", err)
		return
	}

	server.setSessionCookie(writer, result.Token, result.ExpiresAt)
	writeJSON(writer, http.StatusCreated, authResponseBody{
		UserID:   result.User.ID,
		Username: result.User.Username,
	})
}

func (server *Server) handleLogin(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writer.Header().Set("Allow", http.MethodPost)
		writeJSONError(writer, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var payload loginRequest
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		writeJSONError(writer, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := server.app.Login(request.Context(), app.LoginInput{
		Username: payload.Username,
		Password: payload.Password,
	})
	if err != nil {
		server.writeAppError(writer, "login failed", err)
		return
	}

	server.setSessionCookie(writer, result.Token, result.ExpiresAt)
	writeJSON(writer, http.StatusOK, authResponseBody{
		UserID:   result.User.ID,
		Username: result.User.Username,
	})
}

func (server *Server) handleLogout(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writer.Header().Set("Allow", http.MethodPost)
		writeJSONError(writer, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	cookie, err := request.Cookie(sessionCookieName)
	if err == nil {
		if logoutErr := server.app.Logout(request.Context(), cookie.Value); logoutErr != nil {
			server.logger.Warn("logout failed", slog.Any("error", logoutErr))
		}
	}

	clearSessionCookie(writer)
	writer.WriteHeader(http.StatusNoContent)
}

func (server *Server) requireUser(next authenticatedHandler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		cookie, err := request.Cookie(sessionCookieName)
		if err != nil {
			writeJSONError(writer, http.StatusUnauthorized, "authentication required")
			return
		}

		user, err := server.app.CurrentUser(request.Context(), cookie.Value)
		if err != nil {
			server.writeAppError(writer, "session authentication failed", err)
			return
		}

		next(writer, request, user)
	})
}

func (server *Server) setSessionCookie(writer http.ResponseWriter, token string, expiresAt time.Time) {
	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge < 1 {
		maxAge = 1
	}
	http.SetCookie(writer, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearSessionCookie(writer http.ResponseWriter) {
	http.SetCookie(writer, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0).UTC(),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (server *Server) writeAppError(writer http.ResponseWriter, logMessage string, err error) {
	switch {
	case errors.Is(err, app.ErrInvalidInput):
		writeJSONError(writer, http.StatusBadRequest, "invalid input")
	case errors.Is(err, app.ErrUserAlreadyExists):
		writeJSONError(writer, http.StatusConflict, "user already exists")
	case errors.Is(err, app.ErrInvalidCredentials):
		writeJSONError(writer, http.StatusUnauthorized, "invalid credentials")
	default:
		server.logger.Error(logMessage, slog.Any("error", err))
		writeJSONError(writer, http.StatusInternalServerError, "internal server error")
	}
}

func writeJSON(writer http.ResponseWriter, status int, body any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	if err := json.NewEncoder(writer).Encode(body); err != nil {
		slog.Default().Error("failed to encode JSON response", slog.Any("error", err))
	}
}

func writeJSONError(writer http.ResponseWriter, status int, message string) {
	writeJSON(writer, status, map[string]string{"error": message})
}
