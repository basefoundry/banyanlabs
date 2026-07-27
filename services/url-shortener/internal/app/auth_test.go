package app

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/basefoundry/banyanlabs/services/url-shortener/internal/storage/sqlite"
	"golang.org/x/crypto/bcrypt"
)

func TestSignupCreatesUserAndSession(t *testing.T) {
	ctx := context.Background()
	application, db := newAuthTestApp(t, ctx)

	result, err := application.Signup(ctx, SignupInput{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("signup: %v", err)
	}

	if result.User.ID == 0 {
		t.Fatal("user id = 0")
	}
	if result.User.Username != "alice" {
		t.Fatalf("username = %q", result.User.Username)
	}
	if result.User.Email != "alice@example.com" {
		t.Fatalf("email = %q", result.User.Email)
	}
	if result.Token == "" {
		t.Fatal("empty session token")
	}
	if result.Session.UserID != result.User.ID {
		t.Fatalf("session user id = %d, want %d", result.Session.UserID, result.User.ID)
	}
	if !result.ExpiresAt.Equal(result.Session.ExpiresAt) {
		t.Fatalf("result expires_at = %s, session expires_at = %s", result.ExpiresAt, result.Session.ExpiresAt)
	}

	var passwordHash string
	if err := db.QueryRowContext(ctx, "SELECT password_hash FROM users WHERE id = ?", result.User.ID).Scan(&passwordHash); err != nil {
		t.Fatalf("query password hash: %v", err)
	}
	if passwordHash == "correct horse battery staple" {
		t.Fatal("password was stored in plain text")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte("correct horse battery staple")); err != nil {
		t.Fatalf("stored password hash does not verify password: %v", err)
	}

	var tokenHash string
	if err := db.QueryRowContext(ctx, "SELECT token_hash FROM sessions WHERE id = ?", result.Session.ID).Scan(&tokenHash); err != nil {
		t.Fatalf("query token hash: %v", err)
	}
	if tokenHash == result.Token {
		t.Fatal("session token was stored in plain text")
	}
	if tokenHash != hashSessionToken(result.Token) {
		t.Fatal("stored token hash does not match returned session token")
	}
}

func TestSignupRejectsDuplicateUser(t *testing.T) {
	ctx := context.Background()
	application, _ := newAuthTestApp(t, ctx)

	_, err := application.Signup(ctx, SignupInput{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("signup: %v", err)
	}

	_, err = application.Signup(ctx, SignupInput{
		Username: "ALICE",
		Email:    "alice2@example.com",
		Password: "correct horse battery staple",
	})
	if !errors.Is(err, ErrUserAlreadyExists) {
		t.Fatalf("duplicate signup error = %v, want %v", err, ErrUserAlreadyExists)
	}
}

func TestSignupRejectsWeakAuthInputs(t *testing.T) {
	ctx := context.Background()
	application, _ := newAuthTestApp(t, ctx)

	tests := []struct {
		name  string
		input SignupInput
	}{
		{
			name: "short username",
			input: SignupInput{
				Username: "al",
				Email:    "alice@example.com",
				Password: "correct horse battery staple",
			},
		},
		{
			name: "invalid username character",
			input: SignupInput{
				Username: "alice smith",
				Email:    "alice@example.com",
				Password: "correct horse battery staple",
			},
		},
		{
			name: "invalid email",
			input: SignupInput{
				Username: "alice",
				Email:    "alice",
				Password: "correct horse battery staple",
			},
		},
		{
			name: "short password",
			input: SignupInput{
				Username: "alice",
				Email:    "alice@example.com",
				Password: "too short",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := application.Signup(ctx, test.input)
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("signup error = %v, want %v", err, ErrInvalidInput)
			}
		})
	}
}

func TestLoginCreatesSessionForValidCredentials(t *testing.T) {
	ctx := context.Background()
	application, _ := newAuthTestApp(t, ctx)

	signup, err := application.Signup(ctx, SignupInput{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("signup: %v", err)
	}

	login, err := application.Login(ctx, LoginInput{
		Username: "alice",
		Password: "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if login.User.ID != signup.User.ID {
		t.Fatalf("login user id = %d, want %d", login.User.ID, signup.User.ID)
	}
	if login.Token == "" {
		t.Fatal("empty login token")
	}
	if login.Token == signup.Token {
		t.Fatal("login reused signup session token")
	}
}

func TestLoginRejectsInvalidCredentials(t *testing.T) {
	ctx := context.Background()
	application, _ := newAuthTestApp(t, ctx)

	_, err := application.Signup(ctx, SignupInput{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("signup: %v", err)
	}

	_, err = application.Login(ctx, LoginInput{
		Username: "alice",
		Password: "wrong password",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("login error = %v, want %v", err, ErrInvalidCredentials)
	}
}

func TestLoginLocksOutAfterRepeatedFailures(t *testing.T) {
	ctx := context.Background()
	application, _ := newAuthTestApp(t, ctx)

	_, err := application.Signup(ctx, SignupInput{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("signup: %v", err)
	}

	for attempt := 0; attempt < maxFailedLoginAttempts; attempt++ {
		_, err := application.Login(ctx, LoginInput{
			Username: "alice",
			Password: "wrong password",
		})
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("attempt %d error = %v, want %v", attempt+1, err, ErrInvalidCredentials)
		}
	}

	_, err = application.Login(ctx, LoginInput{
		Username: "alice",
		Password: "correct horse battery staple",
	})
	if !errors.Is(err, ErrTooManyAttempts) {
		t.Fatalf("locked login error = %v, want %v", err, ErrTooManyAttempts)
	}
}

func TestLoginClearsFailureCountAfterSuccess(t *testing.T) {
	ctx := context.Background()
	application, _ := newAuthTestApp(t, ctx)

	_, err := application.Signup(ctx, SignupInput{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("signup: %v", err)
	}

	_, err = application.Login(ctx, LoginInput{
		Username: "alice",
		Password: "wrong password",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("first login error = %v, want %v", err, ErrInvalidCredentials)
	}

	if _, err := application.Login(ctx, LoginInput{
		Username: "alice",
		Password: "correct horse battery staple",
	}); err != nil {
		t.Fatalf("successful login: %v", err)
	}

	for attempt := 0; attempt < maxFailedLoginAttempts-1; attempt++ {
		_, err := application.Login(ctx, LoginInput{
			Username: "alice",
			Password: "wrong password",
		})
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("attempt %d error = %v, want %v", attempt+1, err, ErrInvalidCredentials)
		}
	}

	if _, err := application.Login(ctx, LoginInput{
		Username: "alice",
		Password: "correct horse battery staple",
	}); err != nil {
		t.Fatalf("login after cleared failures: %v", err)
	}
}

func TestLogoutDeletesSession(t *testing.T) {
	ctx := context.Background()
	application, db := newAuthTestApp(t, ctx)

	result, err := application.Signup(ctx, SignupInput{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("signup: %v", err)
	}

	if err := application.Logout(ctx, result.Token); err != nil {
		t.Fatalf("logout: %v", err)
	}

	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sessions WHERE id = ?", result.Session.ID).Scan(&count); err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if count != 0 {
		t.Fatalf("session count = %d, want 0", count)
	}
}

func TestCurrentUserReturnsUserForValidSession(t *testing.T) {
	ctx := context.Background()
	application, _ := newAuthTestApp(t, ctx)

	signup, err := application.Signup(ctx, SignupInput{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("signup: %v", err)
	}

	user, err := application.CurrentUser(ctx, signup.Token)
	if err != nil {
		t.Fatalf("current user: %v", err)
	}
	if user.ID != signup.User.ID {
		t.Fatalf("current user id = %d, want %d", user.ID, signup.User.ID)
	}
	if user.Username != "alice" {
		t.Fatalf("current username = %q", user.Username)
	}
}

func TestCurrentUserRejectsLoggedOutSession(t *testing.T) {
	ctx := context.Background()
	application, _ := newAuthTestApp(t, ctx)

	signup, err := application.Signup(ctx, SignupInput{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("signup: %v", err)
	}
	if err := application.Logout(ctx, signup.Token); err != nil {
		t.Fatalf("logout: %v", err)
	}

	_, err = application.CurrentUser(ctx, signup.Token)
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("current user error = %v, want %v", err, ErrInvalidCredentials)
	}
}

func TestCleanupExpiredSessionsDeletesExpiredRows(t *testing.T) {
	ctx := context.Background()
	application, db := newAuthTestApp(t, ctx)
	now := time.Now().UTC()

	signup, err := application.Signup(ctx, SignupInput{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("signup: %v", err)
	}

	login, err := application.Login(ctx, LoginInput{
		Username: "alice",
		Password: "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	expiredAt := now.Add(-time.Minute).Format(time.RFC3339Nano)
	if _, err := db.ExecContext(ctx, "UPDATE sessions SET expires_at = ? WHERE id = ?", expiredAt, signup.Session.ID); err != nil {
		t.Fatalf("expire signup session: %v", err)
	}

	deleted, err := application.CleanupExpiredSessions(ctx, now)
	if err != nil {
		t.Fatalf("cleanup expired sessions: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("deleted sessions = %d, want 1", deleted)
	}

	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sessions WHERE id = ?", signup.Session.ID).Scan(&count); err != nil {
		t.Fatalf("count expired session: %v", err)
	}
	if count != 0 {
		t.Fatalf("expired session count = %d, want 0", count)
	}

	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sessions WHERE id = ?", login.Session.ID).Scan(&count); err != nil {
		t.Fatalf("count active session: %v", err)
	}
	if count != 1 {
		t.Fatalf("active session count = %d, want 1", count)
	}
}

func newAuthTestApp(t *testing.T, ctx context.Context) (*App, *sql.DB) {
	t.Helper()

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

	return New(Options{Store: sqlite.NewStore(db)}), db
}
