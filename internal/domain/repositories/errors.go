package repositories

import "errors"

// ErrNotFound is returned when a queried record does not exist.
// Infrastructure adapters must translate their driver-specific "not found"
// errors (e.g. gorm.ErrRecordNotFound) into this domain-level sentinel
// so that application services can distinguish "not found" from real failures.
var ErrNotFound = errors.New("not found")
