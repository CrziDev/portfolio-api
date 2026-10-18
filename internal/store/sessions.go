package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"
)

// Only a hash of each token is stored, so a copy of the database can't be used to sign in.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// CreateSession stores a new session and clears out expired ones.
func (s *Store) CreateSession(ctx context.Context, token string, expires time.Time) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at <= ?`, time.Now().Unix()); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO sessions (token_hash, expires_at) VALUES (?, ?)`,
		hashToken(token), expires.Unix())
	return err
}

func (s *Store) ValidSession(ctx context.Context, token string) (bool, error) {
	var one int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM sessions WHERE token_hash = ? AND expires_at > ?`,
		hashToken(token), time.Now().Unix()).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (s *Store) DeleteSession(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, hashToken(token))
	return err
}
