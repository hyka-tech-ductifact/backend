package usecases

import (
	"context"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/pagination"

	"github.com/google/uuid"
)

// ProjectService is the inbound port for project operations.
// Inbound adapters (HTTP handlers, CLI, etc.) depend on this interface.
type ProjectService interface {
	CreateProject(ctx context.Context, userID uuid.UUID, params entities.CreateProjectParams) (*entities.Project, error)
	GetProjectByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*entities.Project, error)
	ListProjectsByClientID(
		ctx context.Context,
		clientID uuid.UUID,
		userID uuid.UUID,
		pg pagination.Pagination,
	) (pagination.Result[*entities.Project], error)
	UpdateProject(
		ctx context.Context,
		id uuid.UUID,
		userID uuid.UUID,
		params entities.UpdateProjectParams,
	) (*entities.Project, error)
	DeleteProject(ctx context.Context, id uuid.UUID, userID uuid.UUID, cascade bool) error
}
