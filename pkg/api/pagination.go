package api

// PageResult is a generic paginated response with cursor-based pagination.
type PageResult[T any] struct {
	Items      []T     `json:"items"`
	NextCursor *string `json:"nextCursor,omitempty"`
	HasMore    bool    `json:"hasMore"`
}

// NewPageResult creates a new PageResult.
func NewPageResult[T any](items []T, nextCursor *string, hasMore bool) PageResult[T] {
	return PageResult[T]{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}
}

// CursorFromQuery returns a *string pointer from a raw query parameter value.
// Returns nil if raw is empty, otherwise returns &raw.
func CursorFromQuery(raw string) *string {
	if raw == "" {
		return nil
	}
	return &raw
}
