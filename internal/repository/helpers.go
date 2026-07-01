package repository

import "time"

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
