package persistence

import (
	"fmt"
	"strings"

	domainquery "ductifact/internal/domain/query"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// applyLiteralSearch applies an escaped, case-insensitive substring search.
func applyLiteralSearch(query *gorm.DB, search, predicate string, parameterCount int) *gorm.DB {
	if search == "" {
		return query
	}

	pattern := "%" + escapeLikePattern(search) + "%"
	args := make([]any, parameterCount)
	for i := range args {
		args[i] = pattern
	}
	return query.Where(predicate, args...)
}

func escapeLikePattern(value string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`%`, `\%`,
		`_`, `\_`,
	)
	return replacer.Replace(value)
}

func applyStableOrder(query *gorm.DB, column string, direction domainquery.SortDirection) (*gorm.DB, error) {
	var descending bool
	switch direction {
	case domainquery.SortAscending:
		descending = false
	case domainquery.SortDescending:
		descending = true
	default:
		return nil, fmt.Errorf("invalid sort direction %q", direction)
	}

	return query.Clauses(clause.OrderBy{Columns: []clause.OrderByColumn{
		{Column: clause.Column{Name: column}, Desc: descending},
		{Column: clause.Column{Name: "id"}, Desc: descending},
	}}), nil
}

func applyDefaultPieceDefinitionOrder(query *gorm.DB) *gorm.DB {
	return query.Clauses(clause.OrderBy{Columns: []clause.OrderByColumn{
		{Column: clause.Column{Name: "predefined"}, Desc: true},
		{Column: clause.Column{Name: "created_at"}, Desc: true},
		{Column: clause.Column{Name: "id"}, Desc: true},
	}})
}

func applyPagination(query *gorm.DB, pg domainquery.PageRequest) *gorm.DB {
	return query.Offset(pg.Offset()).Limit(pg.PageSize)
}

func pageStartsAfterResults(pg domainquery.PageRequest, totalItems int64) bool {
	return int64(pg.Offset()) >= totalItems
}
