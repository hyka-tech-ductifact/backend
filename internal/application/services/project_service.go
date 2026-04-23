package services

import (
	"context"
	"errors"
	"time"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/pagination"
	"ductifact/internal/domain/repositories"

	"github.com/google/uuid"
)

var ErrHasAssociatedOrders = errors.New("project has associated orders")

// projectService implements usecases.ProjectService.
// Unexported struct: can only be created via NewProjectService.
type projectService struct {
	projectRepo repositories.ProjectRepository
	clientRepo  repositories.ClientRepository
	orderRepo   repositories.OrderRepository
}

// NewProjectService creates a new ProjectService.
// The client repository is needed to verify ownership during Create and List.
// The order repository is needed to check for associated orders before deletion.
func NewProjectService(projectRepo repositories.ProjectRepository, clientRepo repositories.ClientRepository, orderRepo repositories.OrderRepository) *projectService {
	return &projectService{
		projectRepo: projectRepo,
		clientRepo:  clientRepo,
		orderRepo:   orderRepo,
	}
}

// CreateProject orchestrates project creation:
// 1. Verify the owning client belongs to the user.
// 2. Build the domain entity (which validates all fields).
// 3. Persist via repository.
func (s *projectService) CreateProject(ctx context.Context, userID uuid.UUID, params entities.CreateProjectParams) (*entities.Project, error) {
	// Step 1: Verify client ownership
	_, err := s.clientRepo.GetByIDForOwner(ctx, params.ClientID, userID)
	if err != nil {
		return nil, err
	}

	// Step 2: Domain entity validates its own invariants
	project, err := entities.NewProject(params)
	if err != nil {
		return nil, err
	}

	// Step 3: Persist
	if err := s.projectRepo.Create(ctx, project); err != nil {
		return nil, err
	}

	return project, nil
}

// GetProjectByID retrieves a project by ID, ensuring it belongs to the given user.
func (s *projectService) GetProjectByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*entities.Project, error) {
	return s.projectRepo.GetByIDForOwner(ctx, id, userID)
}

// ListProjectsByClientID retrieves a paginated list of projects for a client,
// ensuring the client belongs to the given user.
func (s *projectService) ListProjectsByClientID(ctx context.Context, clientID uuid.UUID, userID uuid.UUID, pg pagination.Pagination) (pagination.Result[*entities.Project], error) {
	_, err := s.clientRepo.GetByIDForOwner(ctx, clientID, userID)
	if err != nil {
		return pagination.Result[*entities.Project]{}, err
	}

	projects, totalItems, err := s.projectRepo.ListByClientID(ctx, clientID, pg)
	if err != nil {
		return pagination.Result[*entities.Project]{}, err
	}

	return pagination.NewResult(projects, pg, totalItems), nil
}

// UpdateProject applies a partial update to an existing project.
// Only non-nil fields in params are updated. Ensures the project belongs to the given user.
func (s *projectService) UpdateProject(ctx context.Context, id uuid.UUID, userID uuid.UUID, params entities.UpdateProjectParams) (*entities.Project, error) {
	project, err := s.projectRepo.GetByIDForOwner(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	// Nothing to update
	if !params.HasChanges() {
		return project, nil
	}

	if params.Name != nil {
		if err := project.SetName(*params.Name); err != nil {
			return nil, err
		}
	}
	if params.Address != nil {
		if err := project.SetAddress(*params.Address); err != nil {
			return nil, err
		}
	}
	if params.ManagerName != nil {
		if err := project.SetManagerName(*params.ManagerName); err != nil {
			return nil, err
		}
	}
	if params.Phone != nil {
		if err := project.SetPhone(*params.Phone); err != nil {
			return nil, err
		}
	}
	if params.Description != nil {
		if err := project.SetDescription(*params.Description); err != nil {
			return nil, err
		}
	}

	// Update timestamp and persist
	project.UpdatedAt = time.Now()
	if err := s.projectRepo.Update(ctx, project); err != nil {
		return nil, err
	}

	return project, nil
}

// DeleteProject removes a project, ensuring it belongs to the given user.
// If cascade is false and the project has associated orders, it returns ErrHasAssociatedOrders.
func (s *projectService) DeleteProject(ctx context.Context, id uuid.UUID, userID uuid.UUID, cascade bool) error {
	project, err := s.projectRepo.GetByIDForOwner(ctx, id, userID)
	if err != nil {
		return err
	}

	count, err := s.orderRepo.CountByProjectID(ctx, project.ID)
	if err != nil {
		return err
	}

	if count > 0 && !cascade {
		return ErrHasAssociatedOrders
	}

	return s.projectRepo.Delete(ctx, project.ID)
}
