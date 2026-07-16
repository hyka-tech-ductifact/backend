package query

import (
	"ductifact/internal/domain/entities"

	"github.com/google/uuid"
)

// SortDirection is the requested ordering direction for a collection query.
type SortDirection string

const (
	SortAscending  SortDirection = "asc"
	SortDescending SortDirection = "desc"
)

// Sort identifies an API-visible field and its requested direction.
type Sort struct {
	Field     string
	Direction SortDirection
}

// ListQuery contains the controls common to every collection query.
type ListQuery struct {
	Page   PageRequest
	Search string
	Sort   *Sort
}

// ClientListQuery controls retrieval of a user's clients.
type ClientListQuery struct {
	ListQuery
}

// ProjectListQuery controls retrieval of a client's projects.
type ProjectListQuery struct {
	ListQuery
}

// OrderListQuery controls retrieval of a project's orders.
type OrderListQuery struct {
	ListQuery
	Status *entities.OrderStatus
}

// PieceListQuery controls retrieval of an order's pieces.
type PieceListQuery struct {
	ListQuery
	DefinitionID *uuid.UUID
}

// PieceDefinitionListQuery controls retrieval of visible piece definitions.
type PieceDefinitionListQuery struct {
	ListQuery
	Predefined      *bool
	IncludeArchived bool
}
