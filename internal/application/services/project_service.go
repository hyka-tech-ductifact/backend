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

// --- Application-level errors ---

var (
	ErrProjectNotFound = errors.New("project not found")
	ErrProjectNotOwned = errors.New("project does not belong to this client")
)

// projectService implements usecases.ProjectService.
// Unexported struct: can only be created via NewProjectService.
type projectService struct {
	projectRepo repositories.ProjectRepository
	clientRepo  repositories.ClientRepository
}

// NewProjectService creates a new ProjectService.
// It receives the project and client repositories (outbound ports).
// The client repository is needed to verify that the owning client exists
// and belongs to the authenticated user.
func NewProjectService(projectRepo repositories.ProjectRepository, clientRepo repositories.ClientRepository) *projectService {
	return &projectService{
		projectRepo: projectRepo,
		clientRepo:  clientRepo,
	}
}

// CreateProject orchestrates project creation:
// 1. Verify the owning client exists and belongs to the user.
// 2. Build the domain entity (which validates all fields).
// 3. Persist via repository.
func (s *projectService) CreateProject(ctx context.Context, userID uuid.UUID, params entities.CreateProjectParams) (*entities.Project, error) {
	// Step 1: Verify client ownership
	_, err := verifyClientOwnership(ctx, s.clientRepo, params.ClientID, userID)
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

// GetProjectByID retrieves a project by ID, ensuring the client belongs to the user
// and the project belongs to the client.
func (s *projectService) GetProjectByID(ctx context.Context, id uuid.UUID, clientID uuid.UUID, userID uuid.UUID) (*entities.Project, error) {
	return verifyProjectOwnership(ctx, s.clientRepo, s.projectRepo, id, clientID, userID)
}

// ListProjectsByClientID retrieves a paginated list of projects belonging to a client.
// Ensures the client belongs to the authenticated user.
func (s *projectService) ListProjectsByClientID(ctx context.Context, clientID uuid.UUID, userID uuid.UUID, pg pagination.Pagination) (pagination.Result[*entities.Project], error) {
	// Verify ownership chain: User → Client
	_, err := verifyClientOwnership(ctx, s.clientRepo, clientID, userID)
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
// Only non-nil fields in params are updated. Ensures the client belongs to the user
// and the project belongs to the client.
func (s *projectService) UpdateProject(ctx context.Context, id uuid.UUID, clientID uuid.UUID, userID uuid.UUID, params entities.UpdateProjectParams) (*entities.Project, error) {
	project, err := verifyProjectOwnership(ctx, s.clientRepo, s.projectRepo, id, clientID, userID)
	if err != nil {
		return nil, err
	}

	// Nothing to update
	if !params.HasChanges() {
		return project, nil
	}

	// Apply changes
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

// DeleteProject removes a project, ensuring the client belongs to the user
// and the project belongs to the client.
func (s *projectService) DeleteProject(ctx context.Context, id uuid.UUID, clientID uuid.UUID, userID uuid.UUID) error {
	project, err := verifyProjectOwnership(ctx, s.clientRepo, s.projectRepo, id, clientID, userID)
	if err != nil {
		return err
	}

	return s.projectRepo.Delete(ctx, project.ID)
}
