package repository

import "time"

// PaginateCursor processes a slice of items fetched with pageSize+1 limit and
// returns the truncated page, an optional next cursor (ISO-format timestamp of
// the last item), and whether more pages exist.
//
// Use this after executing a cursor-based SELECT with LIMIT pageSize+1.
// cursorExtractor returns the time.Time field used for cursor generation
// (typically CreatedAt or StartedAt).
func PaginateCursor[T any](items []T, pageSize int, cursorExtractor func(T) time.Time) ([]T, *string, bool) {
	hasMore := len(items) > pageSize
	if hasMore {
		items = items[:pageSize]
	}

	var nextCursor *string
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		c := cursorExtractor(last).Format("2006-01-02 15:04:05.999999999")
		nextCursor = &c
	}

	return items, nextCursor, hasMore
}
