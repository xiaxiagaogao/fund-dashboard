package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Friend is a row in the friends table. Both real friends and the operator
// (user themselves, with is_admin=1) live here.
type Friend struct {
	ID           int64
	Name         string
	Username     string
	PasswordHash string
	IsAdmin      bool
	Active       bool
	CreatedAt    int64
}

var ErrFriendNotFound = errors.New("store: friend not found")

func CreateFriend(ctx context.Context, db *sql.DB, name, username, passwordHash string, isAdmin bool) (int64, error) {
	now := time.Now().UnixMilli()
	res, err := db.ExecContext(ctx,
		`INSERT INTO friends(name, username, password_hash, is_admin, created_at) VALUES(?, ?, ?, ?, ?)`,
		name, username, passwordHash, boolToInt(isAdmin), now,
	)
	if err != nil {
		return 0, fmt.Errorf("insert friend: %w", err)
	}
	return res.LastInsertId()
}

func GetFriendByUsername(ctx context.Context, db *sql.DB, username string) (*Friend, error) {
	var f Friend
	var isAdmin, active int
	err := db.QueryRowContext(ctx,
		`SELECT id, name, username, password_hash, is_admin, active, created_at FROM friends WHERE username = ?`,
		username,
	).Scan(&f.ID, &f.Name, &f.Username, &f.PasswordHash, &isAdmin, &active, &f.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrFriendNotFound
	}
	if err != nil {
		return nil, err
	}
	f.IsAdmin = isAdmin != 0
	f.Active = active != 0
	return &f, nil
}

func GetFriendByID(ctx context.Context, db *sql.DB, id int64) (*Friend, error) {
	var f Friend
	var isAdmin, active int
	err := db.QueryRowContext(ctx,
		`SELECT id, name, username, password_hash, is_admin, active, created_at FROM friends WHERE id = ?`,
		id,
	).Scan(&f.ID, &f.Name, &f.Username, &f.PasswordHash, &isAdmin, &active, &f.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrFriendNotFound
	}
	if err != nil {
		return nil, err
	}
	f.IsAdmin = isAdmin != 0
	f.Active = active != 0
	return &f, nil
}

func ListFriends(ctx context.Context, db *sql.DB) ([]Friend, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT id, name, username, password_hash, is_admin, active, created_at FROM friends ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Friend
	for rows.Next() {
		var f Friend
		var isAdmin, active int
		if err := rows.Scan(&f.ID, &f.Name, &f.Username, &f.PasswordHash, &isAdmin, &active, &f.CreatedAt); err != nil {
			return nil, err
		}
		f.IsAdmin = isAdmin != 0
		f.Active = active != 0
		out = append(out, f)
	}
	return out, rows.Err()
}

// SetFriendActive enables (active=true) or disables (active=false) a friend.
// Disabling blocks login and — via the session-validation layer — invalidates
// any live session on the friend's next request. Ledger data is untouched.
func SetFriendActive(ctx context.Context, db *sql.DB, id int64, active bool) error {
	res, err := db.ExecContext(ctx,
		`UPDATE friends SET active = ? WHERE id = ?`, boolToInt(active), id)
	if err != nil {
		return fmt.Errorf("set friend active: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrFriendNotFound
	}
	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
