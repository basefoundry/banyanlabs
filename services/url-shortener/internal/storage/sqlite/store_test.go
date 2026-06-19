package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/basefoundry/banyanlabs/services/url-shortener/internal/storage"
)

func TestStoreCreatesAndFindsUser(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, ctx)
	now := time.Date(2026, 6, 12, 10, 30, 0, 123, time.UTC)

	created, err := store.CreateUser(ctx, storage.CreateUserParams{
		Username:     "alice",
		Email:        "alice@example.com",
		PasswordHash: "hashed-password",
		Now:          now,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	found, err := store.FindUserByUsername(ctx, "ALICE")
	if err != nil {
		t.Fatalf("find user: %v", err)
	}

	if found.ID != created.ID {
		t.Fatalf("found id = %d, want %d", found.ID, created.ID)
	}
	if found.Username != "alice" {
		t.Fatalf("username = %q", found.Username)
	}
	if found.Email != "alice@example.com" {
		t.Fatalf("email = %q", found.Email)
	}
	if found.PasswordHash != "hashed-password" {
		t.Fatalf("password hash = %q", found.PasswordHash)
	}
	if !found.CreatedAt.Equal(now) {
		t.Fatalf("created_at = %s, want %s", found.CreatedAt, now)
	}
	if !found.ModifiedAt.Equal(now) {
		t.Fatalf("modified_at = %s, want %s", found.ModifiedAt, now)
	}
}

func TestStoreRejectsDuplicateUser(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, ctx)
	now := time.Now().UTC()

	_, err := store.CreateUser(ctx, storage.CreateUserParams{
		Username:     "alice",
		Email:        "alice@example.com",
		PasswordHash: "hashed-password",
		Now:          now,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	_, err = store.CreateUser(ctx, storage.CreateUserParams{
		Username:     "ALICE",
		Email:        "alice2@example.com",
		PasswordHash: "hashed-password",
		Now:          now,
	})
	if !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("duplicate username error = %v, want %v", err, storage.ErrConflict)
	}
}

func TestStoreReturnsNotFoundForMissingUser(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, ctx)

	_, err := store.FindUserByUsername(ctx, "missing")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("missing user error = %v, want %v", err, storage.ErrNotFound)
	}
}

func TestStoreCreatesAndDeletesSession(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, ctx)
	now := time.Date(2026, 6, 12, 10, 30, 0, 0, time.UTC)

	user, err := store.CreateUser(ctx, storage.CreateUserParams{
		Username:     "alice",
		Email:        "alice@example.com",
		PasswordHash: "hashed-password",
		Now:          now,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	session, err := store.CreateSession(ctx, storage.CreateSessionParams{
		UserID:    user.ID,
		TokenHash: "hashed-token",
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if session.ID == 0 {
		t.Fatal("session id = 0")
	}
	if session.UserID != user.ID {
		t.Fatalf("session user id = %d, want %d", session.UserID, user.ID)
	}
	if session.TokenHash != "hashed-token" {
		t.Fatalf("session token hash = %q", session.TokenHash)
	}

	if err := store.DeleteSessionByTokenHash(ctx, "hashed-token"); err != nil {
		t.Fatalf("delete session: %v", err)
	}

	_, err = store.CreateSession(ctx, storage.CreateSessionParams{
		UserID:    user.ID,
		TokenHash: "hashed-token",
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("create session after delete: %v", err)
	}
}

func TestStoreFindsAndTouchesSessionByTokenHash(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, ctx)
	now := time.Date(2026, 6, 12, 10, 30, 0, 0, time.UTC)

	user, err := store.CreateUser(ctx, storage.CreateUserParams{
		Username:     "alice",
		Email:        "alice@example.com",
		PasswordHash: "hashed-password",
		Now:          now,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	created, err := store.CreateSession(ctx, storage.CreateSessionParams{
		UserID:    user.ID,
		TokenHash: "hashed-token",
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	found, err := store.FindSessionByTokenHash(ctx, "hashed-token")
	if err != nil {
		t.Fatalf("find session: %v", err)
	}
	if found.ID != created.ID {
		t.Fatalf("found session id = %d, want %d", found.ID, created.ID)
	}
	if found.LastSeenAt != nil {
		t.Fatalf("last_seen_at = %s, want nil", found.LastSeenAt)
	}

	lastSeenAt := now.Add(time.Minute)
	if err := store.TouchSessionByTokenHash(ctx, "hashed-token", lastSeenAt); err != nil {
		t.Fatalf("touch session: %v", err)
	}

	touched, err := store.FindSessionByTokenHash(ctx, "hashed-token")
	if err != nil {
		t.Fatalf("find touched session: %v", err)
	}
	if touched.LastSeenAt == nil {
		t.Fatal("last_seen_at is nil")
	}
	if !touched.LastSeenAt.Equal(lastSeenAt) {
		t.Fatalf("last_seen_at = %s, want %s", touched.LastSeenAt, lastSeenAt)
	}
}

func newTestStore(t *testing.T, ctx context.Context) *Store {
	t.Helper()

	db, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("close database: %v", err)
		}
	})

	if err := RunMigrations(ctx, db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	return NewStore(db)
}
