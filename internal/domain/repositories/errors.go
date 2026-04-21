package repositories

import "errors"

// ErrNotFound is returned when a queried record does not exist.
// Infrastructure adapters must translate their driver-specific "not found"
// errors (e.g. gorm.ErrRecordNotFound) into this domain-level sentinel
// so that application services can distinguish "not found" from real failures.
var ErrNotFound = errors.New("not found")

// --- Ownership-chain errors (returned by ForOwner repository methods) ---
// These provide fine-grained diagnostics for logging while the HTTP layer
// maps them all to a generic 404.

var (
	ErrClientNotFound = errors.New("client not found")
	ErrClientNotOwned = errors.New("client does not belong to this user")

	ErrProjectNotFound = errors.New("project not found")
	ErrProjectNotOwned = errors.New("project does not belong to this user")

	ErrOrderNotFound = errors.New("order not found")
	ErrOrderNotOwned = errors.New("order does not belong to this user")

	ErrPieceNotFound = errors.New("piece not found")
	ErrPieceNotOwned = errors.New("piece does not belong to this user")

	ErrPieceDefNotFound = errors.New("piece definition not found")
	ErrPieceDefNotOwned = errors.New("piece definition does not belong to this user")
)
