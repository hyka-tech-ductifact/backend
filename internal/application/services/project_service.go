package services

import (
	"context"
	"time"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/pagination"
	"ductifact/internal/domain/repositories"

	"github.com/google/uuid"
)

// projectService implements usecases.ProjectService.
type projectService struct {
	projectRepo repositories.ProjectRepository
	clientRepo  repositories.ClientRepository
}

// NewProjectService creates a new ProjectService.
// The client repository is needed to verify ownership during Create.
func NewProjectService(projectRepo repositories.ProjectRepository, clientRepo repositories.ClientRepository) *projectService {
	return &projectService{
		projectRepo: projectRepo,
		clientRepo:  clientRepo,
	}
}

func (s *projectService) CreateProject(ctx context.Context, userID uuid.UUID, params entities.CreateProjectParams) (*entities.Project, error) {
	_, err := s.clientRepo.GetByIDForOwner(ctx, params.ClientID, userID)
	if err != nil {
		return nil, err
	}

	project, err := entities.NewProject(params)
	if err != nil {
		return nil, err
	}

	if err := s.projectRepo.Create(ctx, project); err != nil {
		return nil, err
	}

	return project, nil
}

func (s *projectService) GetProjectByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*entities.Project, error) {
	return s.projectRepo.GetByIDForOwner(ctx, id, userID)
}

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

func (s *projectService) UpdateProject(ctx context.Context, id uuid.UUID, userID uuid.UUID, params entities.UpdateProjectParams) (*entities.Project, error) {
	project, err := s.projectRepo.GetByIDForOwner(ctx, id, userID)
	if err != nil {
		return nil, err
	}

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

	project.UpdatedAt = time.Now()
	if err := s.projectRepo.Update(ctx, project); err != nil {
		return nil, err
	}

	return project, nil
}

func (s *projectService) DeleteProject(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	project, err := s.projectRepo.GetByIDForOwner(ctx, id, userID)
	if err != nil {
		return err
	}

	return s.projectRepo.Delete(ctx, project.ID)
}
