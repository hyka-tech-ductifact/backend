// Package query defines transport-agnostic queries and results for collections.
package query

import (
	"errors"
	"fmt"
)

var (
	// ErrInvalidPage is returned when the page number is less than 1.
	ErrInvalidPage = errors.New("page must be >= 1")

	// ErrInvalidPageSize is returned when the page size is out of range.
	ErrInvalidPageSize = fmt.Errorf("page_size must be between 1 and %d", MaxPageSize)

	// ErrPageOffsetOverflow is returned when page and page_size cannot be
	// represented safely as an int offset.
	ErrPageOffsetOverflow = errors.New("page and page_size produce an offset that is too large")
)

const (
	DefaultPage     = 1
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// PageRequest holds validated, 1-based pagination parameters for a collection query.
type PageRequest struct {
	Page     int
	PageSize int
}

// NewPageRequest creates validated pagination parameters.
func NewPageRequest(page, pageSize int) (PageRequest, error) {
	if page < 1 {
		return PageRequest{}, ErrInvalidPage
	}
	if pageSize < 1 || pageSize > MaxPageSize {
		return PageRequest{}, ErrInvalidPageSize
	}
	maxInt := int(^uint(0) >> 1)
	if page-1 > maxInt/pageSize {
		return PageRequest{}, ErrPageOffsetOverflow
	}
	return PageRequest{Page: page, PageSize: pageSize}, nil
}

// Offset returns the zero-based row offset.
func (p PageRequest) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// PaginatedResult wraps collection data with pagination metadata.
type PaginatedResult[T any] struct {
	Data       []T
	Page       int
	PageSize   int
	TotalItems int64
	TotalPages int64
}

// NewPaginatedResult creates a result with computed pagination metadata.
func NewPaginatedResult[T any](data []T, page PageRequest, totalItems int64) PaginatedResult[T] {
	pageSize := int64(page.PageSize)
	totalPages := totalItems / pageSize
	if totalItems%pageSize > 0 {
		totalPages++
	}
	if data == nil {
		data = []T{}
	}
	return PaginatedResult[T]{
		Data:       data,
		Page:       page.Page,
		PageSize:   page.PageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}
}
