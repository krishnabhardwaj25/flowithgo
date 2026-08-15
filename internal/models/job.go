package models

import (
	"database/sql"
	"time"
)

type Job struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Payload     []byte    `json:"payload"`
	Status      string    `json:"status"`
	Attempts    int       `json:"attempts"`
	MaxAttempts int       `json:"max_attempts"`
	RunAfter    time.Time `json:"run_after"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	LastError   sql.NullString    `json:"last_error"`
}