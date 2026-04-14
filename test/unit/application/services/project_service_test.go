package services_test

import (
	"context"
	"errors"
	"testing"

	"ductifact/internal/application/services"
	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/pagination"
	"ductifact/internal/domain/repositories"
	"ductifact/test/unit/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// CreateProject
// =============================================================================

func TestCreateProject_WithValidData_ReturnsProject(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	svc := services.NewProjectService(&mocks.MockProjectRepository{}, clientRepoReturning(client))

	project, err := svc.CreateProject(context.Background(), userID, entities.CreateProjectParams{
		Name:        "Residential Tower B",
		Address:     "Calle Mayor 12, Madrid",
		ManagerName: "Carlos Pérez",
		Phone:       "+34 699 111 222",
		Description: "14-storey residential building",
		ClientID:    client.ID,
	})

	require.NoError(t, err)
	assert.Equal(t, "Residential Tower B", project.Name)
	assert.Equal(t, "Calle Mayor 12, Madrid", project.Address)
	assert.Equal(t, "Carlos Pérez", project.ManagerName)
	assert.Equal(t, "+34 699 111 222", project.Phone)
	assert.Equal(t, "14-storey residential building", project.Description)
	assert.Equal(t, client.ID, project.ClientID)
}

func TestCreateProject_WithEmptyName_ReturnsError(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	svc := services.NewProjectService(&mocks.MockProjectRepository{}, clientRepoReturning(client))

	project, err := svc.CreateProject(context.Background(), userID, entities.CreateProjectParams{
		ClientID: client.ID,
	})

	assert.Nil(t, project)
	assert.ErrorIs(t, err, entities.ErrEmptyProjectName)
}

func TestCreateProject_WithNonExistingClient_ReturnsError(t *testing.T) {
	svc := services.NewProjectService(&mocks.MockProjectRepository{}, clientRepoReturning(nil))

	project, err := svc.CreateProject(context.Background(), uuid.New(), entities.CreateProjectParams{
		Name:     "Tower B",
		ClientID: uuid.New(),
	})

	assert.Nil(t, project)
	assert.ErrorIs(t, err, services.ErrClientNotFound)
}

func TestCreateProject_WithWrongUser_ReturnsError(t *testing.T) {
	ownerID := uuid.New()
	client := newTestClient(ownerID)
	svc := services.NewProjectService(&mocks.MockProjectRepository{}, clientRepoReturning(client))

	project, err := svc.CreateProject(context.Background(), uuid.New(), entities.CreateProjectParams{
		Name:     "Tower B",
		ClientID: client.ID,
	})

	assert.Nil(t, project)
	assert.ErrorIs(t, err, services.ErrClientNotOwned)
}

func TestCreateProject_WhenRepoFails_ReturnsError(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	projectRepo := &mocks.MockProjectRepository{
		CreateFn: func(ctx context.Context, project *entities.Project) error {
			return errors.New("db connection lost")
		},
	}
	svc := services.NewProjectService(projectRepo, clientRepoReturning(client))

	project, err := svc.CreateProject(context.Background(), userID, entities.CreateProjectParams{
		Name:     "Tower B",
		ClientID: client.ID,
	})

	assert.Nil(t, project)
	assert.EqualError(t, err, "db connection lost")
}

// =============================================================================
// GetProjectByID
// =============================================================================

func TestGetProjectByID_WithExistingProject_ReturnsProject(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	project := newTestProject(client.ID)
	svc := services.NewProjectService(projectRepoReturning(project), clientRepoReturning(client))

	result, err := svc.GetProjectByID(context.Background(), project.ID, client.ID, userID)

	require.NoError(t, err)
	assert.Equal(t, project.Name, result.Name)
	assert.Equal(t, project.ID, result.ID)
}

func TestGetProjectByID_WithNonExistingProject_ReturnsError(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	svc := services.NewProjectService(projectRepoReturning(nil), clientRepoReturning(client))

	result, err := svc.GetProjectByID(context.Background(), uuid.New(), client.ID, userID)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, services.ErrProjectNotFound)
}

func TestGetProjectByID_WithNonExistingClient_ReturnsError(t *testing.T) {
	svc := services.NewProjectService(&mocks.MockProjectRepository{}, clientRepoReturning(nil))

	result, err := svc.GetProjectByID(context.Background(), uuid.New(), uuid.New(), uuid.New())

	assert.Nil(t, result)
	assert.ErrorIs(t, err, services.ErrClientNotFound)
}

func TestGetProjectByID_WithWrongUser_ReturnsError(t *testing.T) {
	ownerID := uuid.New()
	client := newTestClient(ownerID)
	project := newTestProject(client.ID)
	svc := services.NewProjectService(projectRepoReturning(project), clientRepoReturning(client))

	result, err := svc.GetProjectByID(context.Background(), project.ID, client.ID, uuid.New())

	assert.Nil(t, result)
	assert.ErrorIs(t, err, services.ErrClientNotOwned)
}

func TestGetProjectByID_WithWrongClient_ReturnsError(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	otherClientID := uuid.New()
	project := newTestProject(otherClientID) // project belongs to a different client
	svc := services.NewProjectService(projectRepoReturning(project), clientRepoReturning(client))

	result, err := svc.GetProjectByID(context.Background(), project.ID, client.ID, userID)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, services.ErrProjectNotOwned)
}

// =============================================================================
// ListProjectsByClientID
// =============================================================================

func TestListProjectsByClientID_ReturnsProjects(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	expected := []*entities.Project{
		{ID: uuid.New(), Name: "Project 1", ClientID: client.ID},
		{ID: uuid.New(), Name: "Project 2", ClientID: client.ID},
	}
	projectRepo := &mocks.MockProjectRepository{
		ListByClientIDFn: func(ctx context.Context, cID uuid.UUID, pg pagination.Pagination) ([]*entities.Project, int64, error) {
			return expected, 2, nil
		},
	}
	svc := services.NewProjectService(projectRepo, clientRepoReturning(client))

	pg, _ := pagination.NewPagination(1, 20)
	result, err := svc.ListProjectsByClientID(context.Background(), client.ID, userID, pg)

	require.NoError(t, err)
	assert.Len(t, result.Data, 2)
	assert.Equal(t, int64(2), result.TotalItems)
	assert.Equal(t, 1, result.TotalPages)
}

func TestListProjectsByClientID_EmptyList(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	projectRepo := &mocks.MockProjectRepository{
		ListByClientIDFn: func(ctx context.Context, cID uuid.UUID, pg pagination.Pagination) ([]*entities.Project, int64, error) {
			return []*entities.Project{}, 0, nil
		},
	}
	svc := services.NewProjectService(projectRepo, clientRepoReturning(client))

	pg, _ := pagination.NewPagination(1, 20)
	result, err := svc.ListProjectsByClientID(context.Background(), client.ID, userID, pg)

	require.NoError(t, err)
	assert.Empty(t, result.Data)
	assert.Equal(t, int64(0), result.TotalItems)
}

func TestListProjectsByClientID_WithNonExistingClient_ReturnsError(t *testing.T) {
	svc := services.NewProjectService(&mocks.MockProjectRepository{}, clientRepoReturning(nil))

	pg, _ := pagination.NewPagination(1, 20)
	_, err := svc.ListProjectsByClientID(context.Background(), uuid.New(), uuid.New(), pg)

	assert.ErrorIs(t, err, services.ErrClientNotFound)
}

func TestListProjectsByClientID_WithWrongUser_ReturnsError(t *testing.T) {
	ownerID := uuid.New()
	client := newTestClient(ownerID)
	svc := services.NewProjectService(&mocks.MockProjectRepository{}, clientRepoReturning(client))

	pg, _ := pagination.NewPagination(1, 20)
	_, err := svc.ListProjectsByClientID(context.Background(), client.ID, uuid.New(), pg)

	assert.ErrorIs(t, err, services.ErrClientNotOwned)
}

// =============================================================================
// UpdateProject
// =============================================================================

func TestUpdateProject_WithNewName_UpdatesName(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	project := newTestProject(client.ID)
	svc := services.NewProjectService(projectRepoReturning(project), clientRepoReturning(client))

	result, err := svc.UpdateProject(context.Background(), project.ID, client.ID, userID, entities.UpdateProjectParams{
		Name: strPtr("New Name"),
	})

	require.NoError(t, err)
	assert.Equal(t, "New Name", result.Name)
}

func TestUpdateProject_WithAllFields_UpdatesAll(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	project := newTestProject(client.ID)
	svc := services.NewProjectService(projectRepoReturning(project), clientRepoReturning(client))

	result, err := svc.UpdateProject(context.Background(), project.ID, client.ID, userID, entities.UpdateProjectParams{
		Name:        strPtr("Updated Tower"),
		Address:     strPtr("New Address 5"),
		ManagerName: strPtr("New Manager"),
		Phone:       strPtr("+34 600 999 888"),
		Description: strPtr("Updated description"),
	})

	require.NoError(t, err)
	assert.Equal(t, "Updated Tower", result.Name)
	assert.Equal(t, "New Address 5", result.Address)
	assert.Equal(t, "New Manager", result.ManagerName)
	assert.Equal(t, "+34 600 999 888", result.Phone)
	assert.Equal(t, "Updated description", result.Description)
}

func TestUpdateProject_WithEmptyName_ReturnsError(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	project := newTestProject(client.ID)
	svc := services.NewProjectService(projectRepoReturning(project), clientRepoReturning(client))

	result, err := svc.UpdateProject(context.Background(), project.ID, client.ID, userID, entities.UpdateProjectParams{
		Name: strPtr(""),
	})

	assert.Nil(t, result)
	assert.ErrorIs(t, err, entities.ErrEmptyProjectName)
}

func TestUpdateProject_WithWrongUser_ReturnsError(t *testing.T) {
	ownerID := uuid.New()
	client := newTestClient(ownerID)
	project := newTestProject(client.ID)
	svc := services.NewProjectService(projectRepoReturning(project), clientRepoReturning(client))

	result, err := svc.UpdateProject(context.Background(), project.ID, client.ID, uuid.New(), entities.UpdateProjectParams{
		Name: strPtr("New Name"),
	})

	assert.Nil(t, result)
	assert.ErrorIs(t, err, services.ErrClientNotOwned)
}

func TestUpdateProject_WithNonExistingProject_ReturnsError(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	svc := services.NewProjectService(projectRepoReturning(nil), clientRepoReturning(client))

	result, err := svc.UpdateProject(context.Background(), uuid.New(), client.ID, userID, entities.UpdateProjectParams{
		Name: strPtr("New Name"),
	})

	assert.Nil(t, result)
	assert.ErrorIs(t, err, services.ErrProjectNotFound)
}

func TestUpdateProject_UpdatesTimestamp(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	project := newTestProject(client.ID)
	oldTime := project.UpdatedAt
	svc := services.NewProjectService(projectRepoReturning(project), clientRepoReturning(client))

	result, err := svc.UpdateProject(context.Background(), project.ID, client.ID, userID, entities.UpdateProjectParams{
		Name: strPtr("New Name"),
	})

	require.NoError(t, err)
	assert.True(t, result.UpdatedAt.After(oldTime), "UpdatedAt must be newer than the original")
}

func TestUpdateProject_WithNoChanges_SkipsPersistence(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	project := newTestProject(client.ID)
	repo := projectRepoReturning(project)
	updateCalled := false
	repo.UpdateFn = func(ctx context.Context, p *entities.Project) error {
		updateCalled = true
		return nil
	}
	svc := services.NewProjectService(repo, clientRepoReturning(client))

	_, err := svc.UpdateProject(context.Background(), project.ID, client.ID, userID, entities.UpdateProjectParams{})

	require.NoError(t, err)
	assert.False(t, updateCalled, "Update should not be called when no fields change")
}

// =============================================================================
// DeleteProject
// =============================================================================

func TestDeleteProject_WithExistingProject_Succeeds(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	project := newTestProject(client.ID)
	deleteCalled := false
	repo := projectRepoReturning(project)
	repo.DeleteFn = func(ctx context.Context, id uuid.UUID) error {
		deleteCalled = true
		return nil
	}
	svc := services.NewProjectService(repo, clientRepoReturning(client))

	err := svc.DeleteProject(context.Background(), project.ID, client.ID, userID)

	assert.NoError(t, err)
	assert.True(t, deleteCalled)
}

func TestDeleteProject_WithNonExistingProject_ReturnsError(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	svc := services.NewProjectService(projectRepoReturning(nil), clientRepoReturning(client))

	err := svc.DeleteProject(context.Background(), uuid.New(), client.ID, userID)

	assert.ErrorIs(t, err, services.ErrProjectNotFound)
}

func TestDeleteProject_WithWrongUser_ReturnsError(t *testing.T) {
	ownerID := uuid.New()
	client := newTestClient(ownerID)
	project := newTestProject(client.ID)
	svc := services.NewProjectService(projectRepoReturning(project), clientRepoReturning(client))

	err := svc.DeleteProject(context.Background(), project.ID, client.ID, uuid.New())

	assert.ErrorIs(t, err, services.ErrClientNotOwned)
}

func TestDeleteProject_WithNonExistingClient_ReturnsError(t *testing.T) {
	svc := services.NewProjectService(&mocks.MockProjectRepository{}, clientRepoReturning(nil))

	err := svc.DeleteProject(context.Background(), uuid.New(), uuid.New(), uuid.New())

	assert.ErrorIs(t, err, services.ErrClientNotFound)
}

// =============================================================================
// verifyClientOwnership — repo error propagation
// =============================================================================

func TestCreateProject_WhenClientRepoFails_ReturnsError(t *testing.T) {
	clientRepo := &mocks.MockClientRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Client, error) {
			return nil, errors.New("db timeout")
		},
	}
	svc := services.NewProjectService(&mocks.MockProjectRepository{}, clientRepo)

	project, err := svc.CreateProject(context.Background(), uuid.New(), entities.CreateProjectParams{
		Name:     "Tower B",
		ClientID: uuid.New(),
	})

	assert.Nil(t, project)
	assert.EqualError(t, err, "db timeout")
}

// Ensure unused import for repositories is resolved
var _ = repositories.ErrNotFound
