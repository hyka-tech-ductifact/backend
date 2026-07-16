package e2e

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func requirePaginatedData(
	t *testing.T,
	body map[string]any,
	page, pageSize int,
	totalItems, totalPages int,
) []any {
	t.Helper()
	require.Equal(t, float64(page), body["page"])
	require.Equal(t, float64(pageSize), body["page_size"])
	require.Equal(t, float64(totalItems), body["total_items"])
	require.Equal(t, float64(totalPages), body["total_pages"])

	data, ok := body["data"].([]any)
	require.True(t, ok, "data must be an array")
	return data
}
