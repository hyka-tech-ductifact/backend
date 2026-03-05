package entities_test

import (
	"testing"

	"ductifact/internal/domain/entities"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient_WithValidData_ReturnsClient(t *testing.T) {
	userID := uuid.New()
	client, err := entities.NewClient("Acme Corp", userID)

	require.NoError(t, err)
	assert.Equal(t, "Acme Corp", client.Name)
	assert.Equal(t, userID, client.UserID)
	assert.NotEmpty(t, client.ID)
	assert.False(t, client.CreatedAt.IsZero())
	assert.False(t, client.UpdatedAt.IsZero())
}

func TestNewClient_WithEmptyName_ReturnsError(t *testing.T) {
	client, err := entities.NewClient("", uuid.New())

	assert.Nil(t, client)
	assert.ErrorIs(t, err, entities.ErrEmptyClientName)
}

func TestNewClient_GeneratesUniqueIDs(t *testing.T) {
	userID := uuid.New()
	client1, err1 := entities.NewClient("Client 1", userID)
	client2, err2 := entities.NewClient("Client 2", userID)

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEqual(t, client1.ID, client2.ID, "each client must have a unique ID")
}

func TestNewClient_SetsCreatedAtAndUpdatedAtEqual(t *testing.T) {
	client, err := entities.NewClient("Acme Corp", uuid.New())

	require.NoError(t, err)
	assert.Equal(t, client.CreatedAt, client.UpdatedAt,
		"CreatedAt and UpdatedAt must be equal on creation")
}

func TestNewClient_StoresUserID(t *testing.T) {
	userID := uuid.New()
	client, err := entities.NewClient("Acme Corp", userID)

	require.NoError(t, err)
	assert.Equal(t, userID, client.UserID, "client must store the owning user's ID")
}
