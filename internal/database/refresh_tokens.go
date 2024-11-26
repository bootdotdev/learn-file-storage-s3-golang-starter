package database

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	CreateRefreshTokenParams
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	RevokedAt *time.Time `json:"revoked_at"`
}

type CreateRefreshTokenParams struct {
	Token     string    `json:"token"`
	UserID    uuid.UUID `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (c Client) CreateRefreshToken(params CreateRefreshTokenParams) (RefreshToken, error) {
	query := `
		INSERT INTO refresh_tokens (
			token,
			created_at,
			updated_at,
			user_id,
			expires_at
		) VALUES (?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, ?, ?)
	`
	_, err := c.db.Exec(query, params.Token, params.UserID.String(), params.ExpiresAt)
	if err != nil {
		return RefreshToken{}, err
	}

	return c.GetRefreshToken(params.Token)
}

func (c Client) RevokeRefreshToken(token string) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = CURRENT_TIMESTAMP
		WHERE token = ?
	`
	_, err := c.db.Exec(query, token)
	return err
}

func (c Client) GetRefreshToken(token string) (RefreshToken, error) {
	query := `
		SELECT token, created_at, updated_at, user_id, expires_at, revoked_at
		FROM refresh_tokens
		WHERE token = ?
	`
	var rt RefreshToken
	var userID string
	err := c.db.QueryRow(query, token).
		Scan(&rt.Token, &rt.CreatedAt, &rt.UpdatedAt, &userID, &rt.ExpiresAt, &rt.RevokedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return RefreshToken{}, nil
		}
		return RefreshToken{}, err
	}

	rt.UserID, err = uuid.Parse(userID)
	if err != nil {
		return RefreshToken{}, err
	}

	return rt, nil
}

func (c Client) DeleteRefreshToken(token string) error {
	query := `
		DELETE FROM refresh_tokens
		WHERE token = ?
	`
	_, err := c.db.Exec(query, token)
	return err
}
