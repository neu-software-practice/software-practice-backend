package repository

import "time"

// rowScanner is satisfied by *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...interface{}) error
}

// touchTimestamps sets CreatedAt and UpdatedAt to now if they are zero.
func touchTimestamps(createdAt, updatedAt *time.Time) {
	now := time.Now()
	if createdAt != nil && createdAt.IsZero() {
		*createdAt = now
	}
	if updatedAt != nil && updatedAt.IsZero() {
		*updatedAt = now
	}
}
