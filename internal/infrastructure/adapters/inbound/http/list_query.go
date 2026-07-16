package http

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"ductifact/internal/domain/query"

	"github.com/gin-gonic/gin"
)

// maxListSearchCharacters keeps collection searches bounded.
const maxListSearchCharacters = 200

func parseBaseListQuery(c *gin.Context, allowedSortFields map[string]struct{}) (query.ListQuery, error) {
	pg, err := parsePageRequest(c)
	if err != nil {
		return query.ListQuery{}, err
	}

	rawSearch := c.Query("search")
	if !utf8.ValidString(rawSearch) {
		return query.ListQuery{}, fmt.Errorf("search must be valid UTF-8")
	}
	if strings.ContainsRune(rawSearch, '\x00') {
		return query.ListQuery{}, fmt.Errorf("search must not contain NUL characters")
	}
	if utf8.RuneCountInString(rawSearch) > maxListSearchCharacters {
		return query.ListQuery{}, fmt.Errorf("search must not exceed %d characters", maxListSearchCharacters)
	}
	search := strings.TrimSpace(rawSearch)

	base := query.ListQuery{Page: pg, Search: search}
	direction := query.SortAscending
	if rawSortOrder, exists := c.GetQuery("sort_order"); exists {
		switch rawSortOrder {
		case string(query.SortAscending):
			direction = query.SortAscending
		case string(query.SortDescending):
			direction = query.SortDescending
		default:
			return query.ListQuery{}, fmt.Errorf("sort_order must be 'asc' or 'desc'")
		}
	}

	rawSortBy, hasSortBy := c.GetQuery("sort_by")
	if !hasSortBy {
		// A valid sort_order is intentionally ignored unless a sort field is requested.
		return base, nil
	}

	sortBy := rawSortBy
	if sortBy == "" {
		return query.ListQuery{}, fmt.Errorf("sort_by must not be empty")
	}
	if _, allowed := allowedSortFields[sortBy]; !allowed {
		return query.ListQuery{}, fmt.Errorf("invalid sort_by %q", sortBy)
	}

	base.Sort = &query.Sort{Field: sortBy, Direction: direction}
	return base, nil
}

func parsePageRequest(c *gin.Context) (query.PageRequest, error) {
	page, err := parseIntegerQuery(c, "page", query.DefaultPage)
	if err != nil {
		return query.PageRequest{}, err
	}
	pageSize, err := parseIntegerQuery(c, "page_size", query.DefaultPageSize)
	if err != nil {
		return query.PageRequest{}, err
	}
	return query.NewPageRequest(page, pageSize)
}

func parseIntegerQuery(c *gin.Context, name string, defaultValue int) (int, error) {
	raw, exists := c.GetQuery(name)
	if !exists {
		return defaultValue, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", name)
	}
	return value, nil
}

func parseBooleanQuery(c *gin.Context, name string) (*bool, error) {
	raw, exists := c.GetQuery(name)
	if !exists {
		return nil, nil
	}

	var value bool
	switch raw {
	case "true":
		value = true
	case "false":
		value = false
	default:
		return nil, fmt.Errorf("%s must be a boolean (true/false)", name)
	}
	return &value, nil
}
