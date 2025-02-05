package session

import (
	"database/sql"
	"encoding/json"
	"time"
)

type SessionRepositorySqlite struct {
	db *sql.DB
}

func NewSqliteRepository(db *sql.DB) *SessionRepositorySqlite {
	return &SessionRepositorySqlite{
		db: db,
	}
}

type rowscan interface {
	// Scan *sql.Row|Rows.Scan
	Scan(dest ...any) error
}

func (r *SessionRepositorySqlite) scanSessionRow(row rowscan) (*Session, error) {
	var session Session
	var data []byte
	err := row.Scan(
		&session.Id,
		&session.UserId,
		&data,
		&session.CreatedAt,
		&session.ExpiresAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	session.Data = make(map[string]any)
	if err := json.Unmarshal(data, &session.Data); err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *SessionRepositorySqlite) Get(id string) (*Session, error) {
	query := `SELECT id, user_id, data, created_at, expires_at FROM sessions WHERE id = ? AND expires_at > ?`
	return r.scanSessionRow(r.db.QueryRow(query, id, time.Now().UTC()))
}

func (r *SessionRepositorySqlite) GetExpired() ([]Session, error) {
	query := "SELECT id, user_id, data, created_at, expires_at FROM sessions WHERE expires_at < ?"
	rows, err := r.db.Query(query, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sessions []Session
	for rows.Next() {
		s, err := r.scanSessionRow(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, *s)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return sessions, nil
}

func (r *SessionRepositorySqlite) Set(session *Session) error {
	dataJson, err := json.Marshal(session.Data)
	if err != nil {
		return err
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
        UPDATE sessions
        SET user_id = ?,
            data = ?,
            created_at = ?,
            expires_at = ?
        WHERE id = ?`,
        session.UserId,
        dataJson,
        session.CreatedAt,
        session.ExpiresAt,
        session.Id,
    )
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		_, err := tx.Exec(`
            INSERT INTO sessions
            (id, user_id, data, created_at, expires_at)
            VALUES (?, ?, ?, ?, ?)`,
            session.Id,
            session.UserId,
            dataJson,
            session.CreatedAt,
            session.ExpiresAt,
        )
        if err != nil {
            return err
        }
	}

	return tx.Commit()
}

func (r *SessionRepositorySqlite) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM sessions WHERE id = ?`, id)
	return err
}

func (r *SessionRepositorySqlite) GC() error {
	_, err := r.db.Exec(`DELETE FROM sessions WHERE expires_at <= ?`, time.Now().UTC())
	return err
}
