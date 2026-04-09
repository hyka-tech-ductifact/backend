package pagination

import (
	"errors"
	"fmt"
)

var (
	// ErrInvalidPage is returned when the page number is less than 1.
	ErrInvalidPage = errors.New("page must be >= 1")

	// ErrInvalidPageSize is returned when the page size is out of range.
	ErrInvalidPageSize = fmt.Errorf("page_size must be between 1 and %d", MaxPageSize)
)

const (
	// DefaultPage is the page number returned when none is specified.
	DefaultPage = 1

	// DefaultPageSize is the number of items per page when none is specified.
	DefaultPageSize = 20

	// MaxPageSize is the upper bound to prevent clients from requesting
	// excessively large pages that could harm performance.
	MaxPageSize = 100
)

// Pagination holds the pagination parameters extracted from the request.
// Values are already validated and clamped to safe defaults.
type Pagination struct {
	Page     int // 1-based page number
	PageSize int // items per page
}

// NewPagination creates validated pagination parameters.
// It returns an error if page or pageSize are out of valid ranges.
func NewPagination(page, pageSize int) (Pagination, error) {
	if page < 1 {
		return Pagination{}, ErrInvalidPage
	}
	if pageSize < 1 || pageSize > MaxPageSize {
		return Pagination{}, ErrInvalidPageSize
	}
	return Pagination{Page: page, PageSize: pageSize}, nil
}

// Result wraps a paginated response with metadata the client needs
// to navigate between pages.
type Result[T any] struct {
	Data       []T
	Page       int
	PageSize   int
	TotalItems int64
	TotalPages int
}

// NewResult creates a paginated result with computed metadata.
func NewResult[T any](data []T, pg Pagination, totalItems int64) Result[T] {
	totalPages := int(totalItems) / pg.PageSize
	if int(totalItems)%pg.PageSize > 0 {
		totalPages++
	}

	// Ensure data is never nil (return [] instead of null in JSON)
	if data == nil {
		data = []T{}
	}

	return Result[T]{
		Data:       data,
		Page:       pg.Page,
		PageSize:   pg.PageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}
}
