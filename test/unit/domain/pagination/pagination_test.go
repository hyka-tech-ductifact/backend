package pagination_test

import (
	"testing"

	"ductifact/internal/domain/pagination"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// NewPagination — validation
// =============================================================================

func TestNewPagination_ValidValues_ReturnsNoError(t *testing.T) {
	p, err := pagination.NewPagination(2, 10)

	require.NoError(t, err)
	assert.Equal(t, 2, p.Page)
	assert.Equal(t, 10, p.PageSize)
}

func TestNewPagination_MaxPageSize_Allowed(t *testing.T) {
	p, err := pagination.NewPagination(1, pagination.MaxPageSize)

	require.NoError(t, err)
	assert.Equal(t, pagination.MaxPageSize, p.PageSize)
}

func TestNewPagination_ZeroPage_ReturnsError(t *testing.T) {
	_, err := pagination.NewPagination(0, 10)

	assert.ErrorIs(t, err, pagination.ErrInvalidPage)
}

func TestNewPagination_NegativePage_ReturnsError(t *testing.T) {
	_, err := pagination.NewPagination(-5, 10)

	assert.ErrorIs(t, err, pagination.ErrInvalidPage)
}

func TestNewPagination_ZeroPageSize_ReturnsError(t *testing.T) {
	_, err := pagination.NewPagination(1, 0)

	assert.ErrorIs(t, err, pagination.ErrInvalidPageSize)
}

func TestNewPagination_NegativePageSize_ReturnsError(t *testing.T) {
	_, err := pagination.NewPagination(1, -10)

	assert.ErrorIs(t, err, pagination.ErrInvalidPageSize)
}

func TestNewPagination_ExcessivePageSize_ReturnsError(t *testing.T) {
	_, err := pagination.NewPagination(1, 9999)

	assert.ErrorIs(t, err, pagination.ErrInvalidPageSize)
}

// =============================================================================
// NewResult
// =============================================================================

func TestNewResult_CalculatesTotalPages(t *testing.T) {
	data := []string{"a", "b", "c"}
	pg, _ := pagination.NewPagination(1, 2)

	result := pagination.NewResult(data, pg, 5)

	assert.Equal(t, 3, result.TotalPages) // ceil(5/2) = 3
	assert.Equal(t, int64(5), result.TotalItems)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 2, result.PageSize)
}

func TestNewResult_ExactDivision_NoExtraPage(t *testing.T) {
	data := []string{"a", "b"}
	pg, _ := pagination.NewPagination(1, 2)

	result := pagination.NewResult(data, pg, 4)

	assert.Equal(t, 2, result.TotalPages) // 4/2 = 2 exactly
}

func TestNewResult_ZeroItems_ZeroPages(t *testing.T) {
	var data []string
	pg, _ := pagination.NewPagination(1, 20)

	result := pagination.NewResult(data, pg, 0)

	assert.Equal(t, 0, result.TotalPages)
	assert.Equal(t, int64(0), result.TotalItems)
	assert.NotNil(t, result.Data) // should be [] not null
}

func TestNewResult_NilData_ReturnsEmptySlice(t *testing.T) {
	pg, _ := pagination.NewPagination(1, 20)

	result := pagination.NewResult[string](nil, pg, 0)

	assert.NotNil(t, result.Data)
	assert.Empty(t, result.Data)
}

func TestNewResult_SingleItem_OnePage(t *testing.T) {
	data := []string{"only"}
	pg, _ := pagination.NewPagination(1, 20)

	result := pagination.NewResult(data, pg, 1)

	assert.Equal(t, 1, result.TotalPages)
}
