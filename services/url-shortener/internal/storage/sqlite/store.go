package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/basefoundry/banyanlabs/services/url-shortener/internal/storage"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (store *Store) CreateUser(ctx context.Context, params storage.CreateUserParams) (storage.User, error) {
	now := formatTime(params.Now)
	result, err := store.db.ExecContext(
		ctx,
		`INSERT INTO users (username, email, password_hash, created_at, modified_at)
		 VALUES (?, ?, ?, ?, ?)`,
		params.Username,
		params.Email,
		params.PasswordHash,
		now,
		now,
	)
	if err != nil {
		if isConstraintError(err) {
			return storage.User{}, storage.ErrConflict
		}
		return storage.User{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return storage.User{}, fmt.Errorf("read inserted user id: %w", err)
	}
	return store.FindUserByID(ctx, id)
}

func (store *Store) FindUserByID(ctx context.Context, id int64) (storage.User, error) {
	return scanUser(store.db.QueryRowContext(
		ctx,
		`SELECT id, username, email, password_hash, created_at, modified_at
		 FROM users
		 WHERE id = ?`,
		id,
	))
}

func (store *Store) FindUserByUsername(ctx context.Context, username string) (storage.User, error) {
	return scanUser(store.db.QueryRowContext(
		ctx,
		`SELECT id, username, email, password_hash, created_at, modified_at
		 FROM users
		 WHERE username = ? COLLATE NOCASE`,
		username,
	))
}

func (store *Store) CreateSession(ctx context.Context, params storage.CreateSessionParams) (storage.Session, error) {
	result, err := store.db.ExecContext(
		ctx,
		`INSERT INTO sessions (user_id, token_hash, created_at, expires_at)
		 VALUES (?, ?, ?, ?)`,
		params.UserID,
		params.TokenHash,
		formatTime(params.CreatedAt),
		formatTime(params.ExpiresAt),
	)
	if err != nil {
		if isConstraintError(err) {
			return storage.Session{}, storage.ErrConflict
		}
		return storage.Session{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return storage.Session{}, fmt.Errorf("read inserted session id: %w", err)
	}
	return store.findSessionByID(ctx, id)
}

func (store *Store) FindSessionByTokenHash(ctx context.Context, tokenHash string) (storage.Session, error) {
	return scanSession(store.db.QueryRowContext(
		ctx,
		`SELECT id, user_id, token_hash, created_at, expires_at, last_seen_at
		 FROM sessions
		 WHERE token_hash = ?`,
		tokenHash,
	))
}

func (store *Store) TouchSessionByTokenHash(ctx context.Context, tokenHash string, lastSeenAt time.Time) error {
	result, err := store.db.ExecContext(
		ctx,
		"UPDATE sessions SET last_seen_at = ? WHERE token_hash = ?",
		formatTime(lastSeenAt),
		tokenHash,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read touched session row count: %w", err)
	}
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func (store *Store) DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error {
	_, err := store.db.ExecContext(ctx, "DELETE FROM sessions WHERE token_hash = ?", tokenHash)
	return err
}

func (store *Store) DeleteExpiredSessions(ctx context.Context, now time.Time) (int64, error) {
	result, err := store.db.ExecContext(ctx, "DELETE FROM sessions WHERE expires_at <= ?", formatTime(now))
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("read deleted expired session row count: %w", err)
	}
	return rowsAffected, nil
}

func (store *Store) findSessionByID(ctx context.Context, id int64) (storage.Session, error) {
	return scanSession(store.db.QueryRowContext(
		ctx,
		`SELECT id, user_id, token_hash, created_at, expires_at, last_seen_at
		 FROM sessions
		 WHERE id = ?`,
		id,
	))
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(row rowScanner) (storage.User, error) {
	var user storage.User
	var createdAt string
	var modifiedAt string

	err := row.Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &createdAt, &modifiedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return storage.User{}, storage.ErrNotFound
	}
	if err != nil {
		return storage.User{}, err
	}

	user.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return storage.User{}, fmt.Errorf("parse user created_at: %w", err)
	}
	user.ModifiedAt, err = parseTime(modifiedAt)
	if err != nil {
		return storage.User{}, fmt.Errorf("parse user modified_at: %w", err)
	}
	return user, nil
}

func scanSession(row rowScanner) (storage.Session, error) {
	var session storage.Session
	var createdAt string
	var expiresAt string
	var lastSeenAt sql.NullString

	err := row.Scan(&session.ID, &session.UserID, &session.TokenHash, &createdAt, &expiresAt, &lastSeenAt)
	if errors.Is(err, sql.ErrNoRows) {
		return storage.Session{}, storage.ErrNotFound
	}
	if err != nil {
		return storage.Session{}, err
	}

	session.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return storage.Session{}, fmt.Errorf("parse session created_at: %w", err)
	}
	session.ExpiresAt, err = parseTime(expiresAt)
	if err != nil {
		return storage.Session{}, fmt.Errorf("parse session expires_at: %w", err)
	}
	if lastSeenAt.Valid {
		parsedLastSeenAt, err := parseTime(lastSeenAt.String)
		if err != nil {
			return storage.Session{}, fmt.Errorf("parse session last_seen_at: %w", err)
		}
		session.LastSeenAt = &parsedLastSeenAt
	}
	return session, nil
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func parseTime(value string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, value)
}

func isConstraintError(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "constraint")
}
