package repositories

import (
	"context"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/query"

	"github.com/google/uuid"
)

// ProjectRepository is the outbound port for project persistence.
// It is defined in the domain but implemented in infrastructure.
type ProjectRepository interface {
	Create(ctx context.Context, project *entities.Project) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Project, error)
	GetByIDForOwner(ctx context.Context, id uuid.UUID, ownerID uuid.UUID) (*entities.Project, error)
	ListByClientID(
		ctx context.Context,
		clientID uuid.UUID,
		opts query.ProjectListQuery,
	) ([]*entities.Project, int64, error)
	CountByClientID(ctx context.Context, clientID uuid.UUID) (int64, error)
	Update(ctx context.Context, project *entities.Project) error
	Delete(ctx context.Context, id uuid.UUID) error
}
