package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID           int64
	Email        string
	FirstName    string
	LastName     string
	Login        string
	PasswordHash string
}

type BlockSession struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"user_id"`
	StartAt    time.Time `json:"start_at"`
	FinishAt   time.Time `json:"finish_at"`
	BlockRange int64     `json:"block_range"`
}

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) CreateUser(ctx context.Context, u User) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (email, first_name, last_name, login, password_hash)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, u.Email, u.FirstName, u.LastName, u.Login, u.PasswordHash).Scan(&id)
	return id, err
}

func (r *Repository) GetUserByLogin(ctx context.Context, login string) (User, error) {
	var u User
	err := r.pool.QueryRow(ctx, `
		SELECT id, email, first_name, last_name, login, password_hash
		FROM users WHERE login = $1
	`, login).Scan(&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.Login, &u.PasswordHash)
	return u, err
}

func (r *Repository) StartBlockSession(ctx context.Context, userID int64) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO block_sessions (user_id, start_at, block_range)
		VALUES ($1, NOW(), 1)
	`, userID)
	return err
}

func (r *Repository) FinishLastBlockSession(ctx context.Context, userID int64, blockRangeMs int64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE block_sessions
		SET finish_at = NOW(), block_range = $2
		WHERE id = (
			SELECT id FROM block_sessions
			WHERE user_id = $1
			ORDER BY start_at DESC
			LIMIT 1
		)
	`, userID, blockRangeMs)
	return err
}

func (r *Repository) ListBlockSessions(ctx context.Context, userID int64) ([]BlockSession, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, start_at, COALESCE(finish_at, start_at), block_range
		FROM block_sessions
		WHERE user_id = $1
		ORDER BY start_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]BlockSession, 0)
	for rows.Next() {
		var s BlockSession
		if err := rows.Scan(&s.ID, &s.UserID, &s.StartAt, &s.FinishAt, &s.BlockRange); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
