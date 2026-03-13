package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"ductifact/internal/application/services"
	"ductifact/internal/domain/entities"
	"ductifact/test/unit/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// CreateClient
// =============================================================================

func TestCreateClient_WithValidData_ReturnsClient(t *testing.T) {
	userID := uuid.New()
	mockUserRepo := &mocks.MockUserRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.User, error) {
			return &entities.User{ID: userID, Name: "Juan", Email: "juan@example.com"}, nil
		},
	}
	mockClientRepo := &mocks.MockClientRepository{}

	svc := services.NewClientService(mockClientRepo, mockUserRepo)

	client, err := svc.CreateClient(context.Background(), "Acme Corp", userID)

	require.NoError(t, err)
	assert.Equal(t, "Acme Corp", client.Name)
	assert.Equal(t, userID, client.UserID)
	assert.NotEmpty(t, client.ID)
}

func TestCreateClient_WithEmptyName_ReturnsError(t *testing.T) {
	userID := uuid.New()
	mockUserRepo := &mocks.MockUserRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.User, error) {
			return &entities.User{ID: userID}, nil
		},
	}
	mockClientRepo := &mocks.MockClientRepository{}

	svc := services.NewClientService(mockClientRepo, mockUserRepo)

	client, err := svc.CreateClient(context.Background(), "", userID)

	assert.Nil(t, client)
	assert.ErrorIs(t, err, entities.ErrEmptyClientName)
}

func TestCreateClient_WithNonExistingUser_ReturnsError(t *testing.T) {
	mockUserRepo := &mocks.MockUserRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.User, error) {
			return nil, errors.New("not found")
		},
	}
	mockClientRepo := &mocks.MockClientRepository{}

	svc := services.NewClientService(mockClientRepo, mockUserRepo)

	client, err := svc.CreateClient(context.Background(), "Acme Corp", uuid.New())

	assert.Nil(t, client)
	assert.ErrorIs(t, err, services.ErrUserNotFound)
}

func TestCreateClient_WhenRepoFails_ReturnsError(t *testing.T) {
	userID := uuid.New()
	mockUserRepo := &mocks.MockUserRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.User, error) {
			return &entities.User{ID: userID}, nil
		},
	}
	mockClientRepo := &mocks.MockClientRepository{
		CreateFn: func(ctx context.Context, client *entities.Client) error {
			return errors.New("db connection lost")
		},
	}

	svc := services.NewClientService(mockClientRepo, mockUserRepo)

	client, err := svc.CreateClient(context.Background(), "Acme Corp", userID)

	assert.Nil(t, client)
	assert.EqualError(t, err, "db connection lost")
}

// =============================================================================
// GetClientByID
// =============================================================================

func TestGetClientByID_WithExistingClient_ReturnsClient(t *testing.T) {
	userID := uuid.New()
	clientID := uuid.New()
	expectedClient := &entities.Client{
		ID:     clientID,
		Name:   "Acme Corp",
		UserID: userID,
	}

	mockClientRepo := &mocks.MockClientRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Client, error) {
			if id == clientID {
				return expectedClient, nil
			}
			return nil, errors.New("not found")
		},
	}
	mockUserRepo := &mocks.MockUserRepository{}

	svc := services.NewClientService(mockClientRepo, mockUserRepo)

	client, err := svc.GetClientByID(context.Background(), clientID, userID)

	require.NoError(t, err)
	assert.Equal(t, "Acme Corp", client.Name)
	assert.Equal(t, clientID, client.ID)
	assert.Equal(t, userID, client.UserID)
}

func TestGetClientByID_WithNonExistingClient_ReturnsError(t *testing.T) {
	mockClientRepo := &mocks.MockClientRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Client, error) {
			return nil, errors.New("not found")
		},
	}
	mockUserRepo := &mocks.MockUserRepository{}

	svc := services.NewClientService(mockClientRepo, mockUserRepo)

	client, err := svc.GetClientByID(context.Background(), uuid.New(), uuid.New())

	assert.Nil(t, client)
	assert.ErrorIs(t, err, services.ErrClientNotFound)
}

func TestGetClientByID_WithWrongUser_ReturnsError(t *testing.T) {
	ownerID := uuid.New()
	otherUserID := uuid.New()
	clientID := uuid.New()

	mockClientRepo := &mocks.MockClientRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Client, error) {
			return &entities.Client{ID: clientID, Name: "Acme Corp", UserID: ownerID}, nil
		},
	}
	mockUserRepo := &mocks.MockUserRepository{}

	svc := services.NewClientService(mockClientRepo, mockUserRepo)

	client, err := svc.GetClientByID(context.Background(), clientID, otherUserID)

	assert.Nil(t, client)
	assert.ErrorIs(t, err, services.ErrClientNotOwned)
}

// =============================================================================
// ListClientsByUserID
// =============================================================================

func TestListClientsByUserID_ReturnsClients(t *testing.T) {
	userID := uuid.New()
	expectedClients := []*entities.Client{
		{ID: uuid.New(), Name: "Client 1", UserID: userID},
		{ID: uuid.New(), Name: "Client 2", UserID: userID},
	}

	mockClientRepo := &mocks.MockClientRepository{
		ListByUserIDFn: func(ctx context.Context, uid uuid.UUID) ([]*entities.Client, error) {
			return expectedClients, nil
		},
	}
	mockUserRepo := &mocks.MockUserRepository{}

	svc := services.NewClientService(mockClientRepo, mockUserRepo)

	clients, err := svc.ListClientsByUserID(context.Background(), userID)

	require.NoError(t, err)
	assert.Len(t, clients, 2)
	assert.Equal(t, "Client 1", clients[0].Name)
	assert.Equal(t, "Client 2", clients[1].Name)
}

func TestListClientsByUserID_EmptyList(t *testing.T) {
	mockClientRepo := &mocks.MockClientRepository{
		ListByUserIDFn: func(ctx context.Context, uid uuid.UUID) ([]*entities.Client, error) {
			return []*entities.Client{}, nil
		},
	}
	mockUserRepo := &mocks.MockUserRepository{}

	svc := services.NewClientService(mockClientRepo, mockUserRepo)

	clients, err := svc.ListClientsByUserID(context.Background(), uuid.New())

	require.NoError(t, err)
	assert.Empty(t, clients)
}

// =============================================================================
// UpdateClient
// =============================================================================

func TestUpdateClient_WithNewName_UpdatesName(t *testing.T) {
	userID := uuid.New()
	clientID := uuid.New()
	existingClient := &entities.Client{
		ID:        clientID,
		Name:      "Old Name",
		UserID:    userID,
		CreatedAt: time.Now().Add(-time.Hour),
		UpdatedAt: time.Now().Add(-time.Hour),
	}

	mockClientRepo := &mocks.MockClientRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Client, error) {
			cp := *existingClient
			return &cp, nil
		},
		UpdateFn: func(ctx context.Context, client *entities.Client) error {
			return nil
		},
	}
	mockUserRepo := &mocks.MockUserRepository{}

	svc := services.NewClientService(mockClientRepo, mockUserRepo)
	newName := "New Name"

	client, err := svc.UpdateClient(context.Background(), clientID, userID, &newName)

	require.NoError(t, err)
	assert.Equal(t, "New Name", client.Name)
}

func TestUpdateClient_WithEmptyName_ReturnsError(t *testing.T) {
	userID := uuid.New()
	clientID := uuid.New()

	mockClientRepo := &mocks.MockClientRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Client, error) {
			return &entities.Client{ID: clientID, Name: "Acme", UserID: userID}, nil
		},
	}
	mockUserRepo := &mocks.MockUserRepository{}

	svc := services.NewClientService(mockClientRepo, mockUserRepo)
	emptyName := ""

	client, err := svc.UpdateClient(context.Background(), clientID, userID, &emptyName)

	assert.Nil(t, client)
	assert.ErrorIs(t, err, entities.ErrEmptyClientName)
}

func TestUpdateClient_WithWrongUser_ReturnsError(t *testing.T) {
	ownerID := uuid.New()
	otherUserID := uuid.New()
	clientID := uuid.New()

	mockClientRepo := &mocks.MockClientRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Client, error) {
			return &entities.Client{ID: clientID, Name: "Acme", UserID: ownerID}, nil
		},
	}
	mockUserRepo := &mocks.MockUserRepository{}

	svc := services.NewClientService(mockClientRepo, mockUserRepo)
	newName := "New Name"

	client, err := svc.UpdateClient(context.Background(), clientID, otherUserID, &newName)

	assert.Nil(t, client)
	assert.ErrorIs(t, err, services.ErrClientNotOwned)
}

func TestUpdateClient_WithNonExistingClient_ReturnsError(t *testing.T) {
	mockClientRepo := &mocks.MockClientRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Client, error) {
			return nil, errors.New("not found")
		},
	}
	mockUserRepo := &mocks.MockUserRepository{}

	svc := services.NewClientService(mockClientRepo, mockUserRepo)
	newName := "New Name"

	client, err := svc.UpdateClient(context.Background(), uuid.New(), uuid.New(), &newName)

	assert.Nil(t, client)
	assert.ErrorIs(t, err, services.ErrClientNotFound)
}

func TestUpdateClient_UpdatesTimestamp(t *testing.T) {
	userID := uuid.New()
	clientID := uuid.New()
	oldTime := time.Now().Add(-time.Hour)
	existingClient := &entities.Client{
		ID:        clientID,
		Name:      "Old Name",
		UserID:    userID,
		CreatedAt: oldTime,
		UpdatedAt: oldTime,
	}

	mockClientRepo := &mocks.MockClientRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Client, error) {
			cp := *existingClient
			return &cp, nil
		},
		UpdateFn: func(ctx context.Context, client *entities.Client) error {
			return nil
		},
	}
	mockUserRepo := &mocks.MockUserRepository{}

	svc := services.NewClientService(mockClientRepo, mockUserRepo)
	newName := "New Name"

	client, err := svc.UpdateClient(context.Background(), clientID, userID, &newName)

	require.NoError(t, err)
	assert.True(t, client.UpdatedAt.After(oldTime), "UpdatedAt must be newer than the original")
}

// =============================================================================
// DeleteClient
// =============================================================================

func TestDeleteClient_WithExistingClient_Succeeds(t *testing.T) {
	userID := uuid.New()
	clientID := uuid.New()
	deleteCalled := false

	mockClientRepo := &mocks.MockClientRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Client, error) {
			return &entities.Client{ID: clientID, Name: "Acme", UserID: userID}, nil
		},
		DeleteFn: func(ctx context.Context, id uuid.UUID) error {
			deleteCalled = true
			return nil
		},
	}
	mockUserRepo := &mocks.MockUserRepository{}

	svc := services.NewClientService(mockClientRepo, mockUserRepo)

	err := svc.DeleteClient(context.Background(), clientID, userID)

	assert.NoError(t, err)
	assert.True(t, deleteCalled, "Delete should have been called on the repository")
}

func TestDeleteClient_WithNonExistingClient_ReturnsError(t *testing.T) {
	mockClientRepo := &mocks.MockClientRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Client, error) {
			return nil, errors.New("not found")
		},
	}
	mockUserRepo := &mocks.MockUserRepository{}

	svc := services.NewClientService(mockClientRepo, mockUserRepo)

	err := svc.DeleteClient(context.Background(), uuid.New(), uuid.New())

	assert.ErrorIs(t, err, services.ErrClientNotFound)
}

func TestDeleteClient_WithWrongUser_ReturnsError(t *testing.T) {
	ownerID := uuid.New()
	otherUserID := uuid.New()
	clientID := uuid.New()

	mockClientRepo := &mocks.MockClientRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Client, error) {
			return &entities.Client{ID: clientID, Name: "Acme", UserID: ownerID}, nil
		},
	}
	mockUserRepo := &mocks.MockUserRepository{}

	svc := services.NewClientService(mockClientRepo, mockUserRepo)

	err := svc.DeleteClient(context.Background(), clientID, otherUserID)

	assert.ErrorIs(t, err, services.ErrClientNotOwned)
}
