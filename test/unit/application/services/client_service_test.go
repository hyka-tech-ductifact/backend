package services_test

import (
	"context"
	"errors"
	"testing"

	"ductifact/internal/application/services"
	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/pagination"
	"ductifact/test/unit/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// CreateClient
// =============================================================================

func TestCreateClient_WithValidData_ReturnsClient(t *testing.T) {
	user := newTestUser()
	svc := services.NewClientService(&mocks.MockClientRepository{}, userRepoReturning(user))

	client, err := svc.CreateClient(context.Background(), "Acme Corp", user.ID)

	require.NoError(t, err)
	assert.Equal(t, "Acme Corp", client.Name)
	assert.Equal(t, user.ID, client.UserID)
}

func TestCreateClient_WithEmptyName_ReturnsError(t *testing.T) {
	user := newTestUser()
	svc := services.NewClientService(&mocks.MockClientRepository{}, userRepoReturning(user))

	client, err := svc.CreateClient(context.Background(), "", user.ID)

	assert.Nil(t, client)
	assert.ErrorIs(t, err, entities.ErrEmptyClientName)
}

func TestCreateClient_WithNonExistingUser_ReturnsError(t *testing.T) {
	svc := services.NewClientService(&mocks.MockClientRepository{}, userRepoReturning(nil))

	client, err := svc.CreateClient(context.Background(), "Acme Corp", uuid.New())

	assert.Nil(t, client)
	assert.ErrorIs(t, err, services.ErrUserNotFound)
}

func TestCreateClient_WhenRepoFails_ReturnsError(t *testing.T) {
	user := newTestUser()
	clientRepo := &mocks.MockClientRepository{
		CreateFn: func(ctx context.Context, client *entities.Client) error {
			return errors.New("db connection lost")
		},
	}
	svc := services.NewClientService(clientRepo, userRepoReturning(user))

	client, err := svc.CreateClient(context.Background(), "Acme Corp", user.ID)

	assert.Nil(t, client)
	assert.EqualError(t, err, "db connection lost")
}

// =============================================================================
// GetClientByID
// =============================================================================

func TestGetClientByID_WithExistingClient_ReturnsClient(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	svc := services.NewClientService(clientRepoReturning(client), &mocks.MockUserRepository{})

	result, err := svc.GetClientByID(context.Background(), client.ID, userID)

	require.NoError(t, err)
	assert.Equal(t, client.Name, result.Name)
	assert.Equal(t, client.ID, result.ID)
}

func TestGetClientByID_WithNonExistingClient_ReturnsError(t *testing.T) {
	svc := services.NewClientService(clientRepoReturning(nil), &mocks.MockUserRepository{})

	result, err := svc.GetClientByID(context.Background(), uuid.New(), uuid.New())

	assert.Nil(t, result)
	assert.ErrorIs(t, err, services.ErrClientNotFound)
}

func TestGetClientByID_WithWrongUser_ReturnsError(t *testing.T) {
	ownerID := uuid.New()
	client := newTestClient(ownerID)
	svc := services.NewClientService(clientRepoReturning(client), &mocks.MockUserRepository{})

	result, err := svc.GetClientByID(context.Background(), client.ID, uuid.New())

	assert.Nil(t, result)
	assert.ErrorIs(t, err, services.ErrClientNotOwned)
}

// =============================================================================
// ListClientsByUserID
// =============================================================================

func TestListClientsByUserID_ReturnsClients(t *testing.T) {
	userID := uuid.New()
	expected := []*entities.Client{
		{ID: uuid.New(), Name: "Client 1", UserID: userID},
		{ID: uuid.New(), Name: "Client 2", UserID: userID},
	}
	clientRepo := &mocks.MockClientRepository{
		ListByUserIDFn: func(ctx context.Context, uid uuid.UUID, pg pagination.Pagination) ([]*entities.Client, int64, error) {
			return expected, 2, nil
		},
	}
	svc := services.NewClientService(clientRepo, &mocks.MockUserRepository{})

	pg, _ := pagination.NewPagination(1, 20)
	result, err := svc.ListClientsByUserID(context.Background(), userID, pg)

	require.NoError(t, err)
	assert.Len(t, result.Data, 2)
	assert.Equal(t, int64(2), result.TotalItems)
	assert.Equal(t, 1, result.TotalPages)
}

func TestListClientsByUserID_EmptyList(t *testing.T) {
	clientRepo := &mocks.MockClientRepository{
		ListByUserIDFn: func(ctx context.Context, uid uuid.UUID, pg pagination.Pagination) ([]*entities.Client, int64, error) {
			return []*entities.Client{}, 0, nil
		},
	}
	svc := services.NewClientService(clientRepo, &mocks.MockUserRepository{})

	pg, _ := pagination.NewPagination(1, 20)
	result, err := svc.ListClientsByUserID(context.Background(), uuid.New(), pg)

	require.NoError(t, err)
	assert.Empty(t, result.Data)
	assert.Equal(t, int64(0), result.TotalItems)
}

// =============================================================================
// UpdateClient
// =============================================================================

func TestUpdateClient_WithNewName_UpdatesName(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	svc := services.NewClientService(clientRepoReturning(client), &mocks.MockUserRepository{})

	result, err := svc.UpdateClient(context.Background(), client.ID, userID, strPtr("New Name"))

	require.NoError(t, err)
	assert.Equal(t, "New Name", result.Name)
}

func TestUpdateClient_WithEmptyName_ReturnsError(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	svc := services.NewClientService(clientRepoReturning(client), &mocks.MockUserRepository{})

	result, err := svc.UpdateClient(context.Background(), client.ID, userID, strPtr(""))

	assert.Nil(t, result)
	assert.ErrorIs(t, err, entities.ErrEmptyClientName)
}

func TestUpdateClient_WithWrongUser_ReturnsError(t *testing.T) {
	ownerID := uuid.New()
	client := newTestClient(ownerID)
	svc := services.NewClientService(clientRepoReturning(client), &mocks.MockUserRepository{})

	result, err := svc.UpdateClient(context.Background(), client.ID, uuid.New(), strPtr("New Name"))

	assert.Nil(t, result)
	assert.ErrorIs(t, err, services.ErrClientNotOwned)
}

func TestUpdateClient_WithNonExistingClient_ReturnsError(t *testing.T) {
	svc := services.NewClientService(clientRepoReturning(nil), &mocks.MockUserRepository{})

	result, err := svc.UpdateClient(context.Background(), uuid.New(), uuid.New(), strPtr("New Name"))

	assert.Nil(t, result)
	assert.ErrorIs(t, err, services.ErrClientNotFound)
}

func TestUpdateClient_UpdatesTimestamp(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	oldTime := client.UpdatedAt
	svc := services.NewClientService(clientRepoReturning(client), &mocks.MockUserRepository{})

	result, err := svc.UpdateClient(context.Background(), client.ID, userID, strPtr("New Name"))

	require.NoError(t, err)
	assert.True(t, result.UpdatedAt.After(oldTime), "UpdatedAt must be newer than the original")
}

func TestUpdateClient_WithNoChanges_SkipsPersistence(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	repo := clientRepoReturning(client)
	updateCalled := false
	repo.UpdateFn = func(ctx context.Context, c *entities.Client) error {
		updateCalled = true
		return nil
	}
	svc := services.NewClientService(repo, &mocks.MockUserRepository{})

	_, err := svc.UpdateClient(context.Background(), client.ID, userID, nil)

	require.NoError(t, err)
	assert.False(t, updateCalled, "Update should not be called when no fields change")
}

// =============================================================================
// DeleteClient
// =============================================================================

func TestDeleteClient_WithExistingClient_Succeeds(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	deleteCalled := false
	repo := clientRepoReturning(client)
	repo.DeleteFn = func(ctx context.Context, id uuid.UUID) error {
		deleteCalled = true
		return nil
	}
	svc := services.NewClientService(repo, &mocks.MockUserRepository{})

	err := svc.DeleteClient(context.Background(), client.ID, userID)

	assert.NoError(t, err)
	assert.True(t, deleteCalled)
}

func TestDeleteClient_WithNonExistingClient_ReturnsError(t *testing.T) {
	svc := services.NewClientService(clientRepoReturning(nil), &mocks.MockUserRepository{})

	err := svc.DeleteClient(context.Background(), uuid.New(), uuid.New())

	assert.ErrorIs(t, err, services.ErrClientNotFound)
}

func TestDeleteClient_WithWrongUser_ReturnsError(t *testing.T) {
	ownerID := uuid.New()
	client := newTestClient(ownerID)
	svc := services.NewClientService(clientRepoReturning(client), &mocks.MockUserRepository{})

	err := svc.DeleteClient(context.Background(), client.ID, uuid.New())

	assert.ErrorIs(t, err, services.ErrClientNotOwned)
}
