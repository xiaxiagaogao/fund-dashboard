package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// WriteAudit records one append-only audit row. payload is JSON-encoded.
// Used for: cash_event.create, login, snapshot.write, friend.create, ...
func WriteAudit(ctx context.Context, db *sql.DB, actor, action string, payload any) error {
	buf, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal audit payload: %w", err)
	}
	_, err = db.ExecContext(ctx,
		`INSERT INTO audit_log(actor, action, payload_json, created_at) VALUES(?, ?, ?, ?)`,
		actor, action, string(buf), time.Now().UnixMilli(),
	)
	if err != nil {
		return fmt.Errorf("insert audit_log: %w", err)
	}
	return nil
}
