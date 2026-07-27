package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/basefoundry/banyanlabs/services/url-shortener/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

const (
	sessionTokenBytes     = 32
	sessionTTL            = 24 * time.Hour
	sessionReaperInterval = time.Hour
)

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrStorageUnavailable = errors.New("storage unavailable")
)

type SignupInput struct {
	Username string
	Email    string
	Password string
}

type LoginInput struct {
	Username string
	Password string
}

type AuthResult struct {
	User      User
	Session   Session
	Token     string
	ExpiresAt time.Time
}

type User struct {
	ID         int64
	Username   string
	Email      string
	CreatedAt  time.Time
	ModifiedAt time.Time
}

type Session struct {
	ID        int64
	UserID    int64
	CreatedAt time.Time
	ExpiresAt time.Time
}

func (app *App) Signup(ctx context.Context, input SignupInput) (AuthResult, error) {
	username := strings.TrimSpace(input.Username)
	email := strings.TrimSpace(input.Email)
	if username == "" || email == "" || input.Password == "" {
		return AuthResult{}, ErrInvalidInput
	}
	if !strings.Contains(email, "@") {
		return AuthResult{}, ErrInvalidInput
	}

	store, err := app.requireStore()
	if err != nil {
		return AuthResult{}, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return AuthResult{}, fmt.Errorf("hash password: %w", err)
	}

	now := time.Now().UTC()
	user, err := store.CreateUser(ctx, storage.CreateUserParams{
		Username:     username,
		Email:        email,
		PasswordHash: string(passwordHash),
		Now:          now,
	})
	if err != nil {
		if errors.Is(err, storage.ErrConflict) {
			return AuthResult{}, ErrUserAlreadyExists
		}
		return AuthResult{}, fmt.Errorf("create user: %w", err)
	}

	return app.createSession(ctx, user)
}

func (app *App) Login(ctx context.Context, input LoginInput) (AuthResult, error) {
	username := strings.TrimSpace(input.Username)
	if username == "" || input.Password == "" {
		return AuthResult{}, ErrInvalidInput
	}

	store, err := app.requireStore()
	if err != nil {
		return AuthResult{}, err
	}

	user, err := store.FindUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return AuthResult{}, ErrInvalidCredentials
		}
		return AuthResult{}, fmt.Errorf("find user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return AuthResult{}, ErrInvalidCredentials
	}

	return app.createSession(ctx, user)
}

func (app *App) Logout(ctx context.Context, token string) error {
	if strings.TrimSpace(token) == "" {
		return ErrInvalidInput
	}

	store, err := app.requireStore()
	if err != nil {
		return err
	}

	if err := store.DeleteSessionByTokenHash(ctx, hashSessionToken(token)); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (app *App) CurrentUser(ctx context.Context, token string) (User, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return User{}, ErrInvalidCredentials
	}

	store, err := app.requireStore()
	if err != nil {
		return User{}, err
	}

	tokenHash := hashSessionToken(token)
	session, err := store.FindSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return User{}, ErrInvalidCredentials
		}
		return User{}, fmt.Errorf("find session: %w", err)
	}

	now := time.Now().UTC()
	if !session.ExpiresAt.After(now) {
		if err := store.DeleteSessionByTokenHash(ctx, tokenHash); err != nil {
			return User{}, fmt.Errorf("delete expired session: %w", err)
		}
		return User{}, ErrInvalidCredentials
	}

	user, err := store.FindUserByID(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return User{}, ErrInvalidCredentials
		}
		return User{}, fmt.Errorf("find session user: %w", err)
	}

	if err := store.TouchSessionByTokenHash(ctx, tokenHash, now); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return User{}, ErrInvalidCredentials
		}
		return User{}, fmt.Errorf("touch session: %w", err)
	}

	return publicUser(user), nil
}

func (app *App) CleanupExpiredSessions(ctx context.Context, now time.Time) (int64, error) {
	store, err := app.requireStore()
	if err != nil {
		return 0, err
	}

	deleted, err := store.DeleteExpiredSessions(ctx, now.UTC())
	if err != nil {
		return 0, fmt.Errorf("delete expired sessions: %w", err)
	}
	return deleted, nil
}

func (app *App) StartSessionReaper(ctx context.Context) {
	app.startSessionReaper(ctx, sessionReaperInterval)
}

func (app *App) startSessionReaper(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = sessionReaperInterval
	}

	go func() {
		app.reapExpiredSessions(ctx)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				app.reapExpiredSessions(ctx)
			}
		}
	}()
}

func (app *App) reapExpiredSessions(ctx context.Context) {
	deleted, err := app.CleanupExpiredSessions(ctx, time.Now().UTC())
	if err != nil {
		if ctx.Err() == nil {
			app.logger.Warn("expired session cleanup failed", slog.Any("error", err))
		}
		return
	}

	if deleted > 0 {
		app.logger.Info("expired sessions deleted", slog.Int64("count", deleted))
	}
}

func (app *App) requireStore() (storage.Store, error) {
	if app.store == nil {
		return nil, ErrStorageUnavailable
	}
	return app.store, nil
}

func (app *App) createSession(ctx context.Context, user storage.User) (AuthResult, error) {
	store, err := app.requireStore()
	if err != nil {
		return AuthResult{}, err
	}

	token, err := newSessionToken()
	if err != nil {
		return AuthResult{}, fmt.Errorf("create session token: %w", err)
	}

	now := time.Now().UTC()
	expiresAt := now.Add(sessionTTL)
	session, err := store.CreateSession(ctx, storage.CreateSessionParams{
		UserID:    user.ID,
		TokenHash: hashSessionToken(token),
		CreatedAt: now,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return AuthResult{}, fmt.Errorf("create session: %w", err)
	}

	return AuthResult{
		User:      publicUser(user),
		Session:   publicSession(session),
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

func publicUser(user storage.User) User {
	return User{
		ID:         user.ID,
		Username:   user.Username,
		Email:      user.Email,
		CreatedAt:  user.CreatedAt,
		ModifiedAt: user.ModifiedAt,
	}
}

func publicSession(session storage.Session) Session {
	return Session{
		ID:        session.ID,
		UserID:    session.UserID,
		CreatedAt: session.CreatedAt,
		ExpiresAt: session.ExpiresAt,
	}
}

func newSessionToken() (string, error) {
	var bytes [sessionTokenBytes]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes[:]), nil
}

func hashSessionToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
