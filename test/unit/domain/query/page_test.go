package query_test

import (
	"testing"

	"ductifact/internal/domain/query"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// PageRequest — validation
// =============================================================================

func TestNewPageRequest_ValidValues_ReturnsNoError(t *testing.T) {
	p, err := query.NewPageRequest(2, 10)

	require.NoError(t, err)
	assert.Equal(t, 2, p.Page)
	assert.Equal(t, 10, p.PageSize)
}

func TestNewPageRequest_MaxPageSize_Allowed(t *testing.T) {
	p, err := query.NewPageRequest(1, query.MaxPageSize)

	require.NoError(t, err)
	assert.Equal(t, query.MaxPageSize, p.PageSize)
}

func TestNewPageRequest_ZeroPage_ReturnsError(t *testing.T) {
	_, err := query.NewPageRequest(0, 10)

	assert.ErrorIs(t, err, query.ErrInvalidPage)
}

func TestNewPageRequest_NegativePage_ReturnsError(t *testing.T) {
	_, err := query.NewPageRequest(-5, 10)

	assert.ErrorIs(t, err, query.ErrInvalidPage)
}

func TestNewPageRequest_ZeroPageSize_ReturnsError(t *testing.T) {
	_, err := query.NewPageRequest(1, 0)

	assert.ErrorIs(t, err, query.ErrInvalidPageSize)
}

func TestNewPageRequest_NegativePageSize_ReturnsError(t *testing.T) {
	_, err := query.NewPageRequest(1, -10)

	assert.ErrorIs(t, err, query.ErrInvalidPageSize)
}

func TestNewPageRequest_ExcessivePageSize_ReturnsError(t *testing.T) {
	_, err := query.NewPageRequest(1, 9999)

	assert.ErrorIs(t, err, query.ErrInvalidPageSize)
}

func TestPageRequest_Offset_ReturnsZeroBasedRowOffset(t *testing.T) {
	p, err := query.NewPageRequest(3, 25)

	require.NoError(t, err)
	assert.Equal(t, 50, p.Offset())
}

func TestNewPageRequest_OffsetOverflow_ReturnsError(t *testing.T) {
	maxInt := int(^uint(0) >> 1)

	_, err := query.NewPageRequest(maxInt, query.MaxPageSize)

	assert.ErrorIs(t, err, query.ErrPageOffsetOverflow)
}

func TestNewPageRequest_LargeSafeOffset_ReturnsOffset(t *testing.T) {
	maxInt := int(^uint(0) >> 1)

	p, err := query.NewPageRequest(maxInt, 1)

	require.NoError(t, err)
	assert.Equal(t, maxInt-1, p.Offset())
}

// =============================================================================
// PaginatedResult
// =============================================================================

func TestNewPaginatedResult_CalculatesTotalPages(t *testing.T) {
	data := []string{"a", "b", "c"}
	pg, _ := query.NewPageRequest(1, 2)

	result := query.NewPaginatedResult(data, pg, 5)

	assert.Equal(t, int64(3), result.TotalPages) // ceil(5/2) = 3
	assert.Equal(t, int64(5), result.TotalItems)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 2, result.PageSize)
}

func TestNewPaginatedResult_ExactDivision_NoExtraPage(t *testing.T) {
	data := []string{"a", "b"}
	pg, _ := query.NewPageRequest(1, 2)

	result := query.NewPaginatedResult(data, pg, 4)

	assert.Equal(t, int64(2), result.TotalPages) // 4/2 = 2 exactly
}

func TestNewPaginatedResult_ZeroItems_ZeroPages(t *testing.T) {
	var data []string
	pg, _ := query.NewPageRequest(1, 20)

	result := query.NewPaginatedResult(data, pg, 0)

	assert.Equal(t, int64(0), result.TotalPages)
	assert.Equal(t, int64(0), result.TotalItems)
	assert.NotNil(t, result.Data) // should be [] not null
}

func TestNewPaginatedResult_NilData_ReturnsEmptySlice(t *testing.T) {
	pg, _ := query.NewPageRequest(1, 20)

	result := query.NewPaginatedResult[string](nil, pg, 0)

	assert.NotNil(t, result.Data)
	assert.Empty(t, result.Data)
}

func TestNewPaginatedResult_SingleItem_OnePage(t *testing.T) {
	data := []string{"only"}
	pg, _ := query.NewPageRequest(1, 20)

	result := query.NewPaginatedResult(data, pg, 1)

	assert.Equal(t, int64(1), result.TotalPages)
}

func TestNewPaginatedResult_LargeTotal_DoesNotOverflowInt32(t *testing.T) {
	pg, _ := query.NewPageRequest(1, 1)
	totalItems := int64(1<<32) + 1

	result := query.NewPaginatedResult[string](nil, pg, totalItems)

	assert.Equal(t, totalItems, result.TotalPages)
}
