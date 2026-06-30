package api

// PageResponse is a generic page-based paginated response.
type PageResponse[T any] struct {
	Items    []T `json:"items"`
	Total    int `json:"total"`
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
}

// NewPageResponse creates a new PageResponse.
func NewPageResponse[T any](items []T, total, page, pageSize int) PageResponse[T] {
	return PageResponse[T]{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}
}
