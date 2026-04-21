package entities_test

import (
	"strings"
	"testing"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/valueobjects"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient_WithValidData_ReturnsClient(t *testing.T) {
	userID := uuid.New()
	client, err := entities.NewClient(entities.CreateClientParams{
		Name:        "Acme Corp",
		Phone:       "+34 612 345 678",
		Email:       "contact@acme.com",
		Description: "Main partner",
		UserID:      userID,
	})

	require.NoError(t, err)
	assert.Equal(t, "Acme Corp", client.Name)
	assert.Equal(t, "+34 612 345 678", client.Phone)
	assert.Equal(t, "contact@acme.com", client.Email)
	assert.Equal(t, "Main partner", client.Description)
	assert.Equal(t, userID, client.UserID)
	assert.NotEmpty(t, client.ID)
	assert.False(t, client.CreatedAt.IsZero())
	assert.False(t, client.UpdatedAt.IsZero())
}

func TestNewClient_WithOptionalFieldsEmpty_ReturnsClient(t *testing.T) {
	userID := uuid.New()
	client, err := entities.NewClient(entities.CreateClientParams{
		Name:   "Acme Corp",
		UserID: userID,
	})

	require.NoError(t, err)
	assert.Equal(t, "Acme Corp", client.Name)
	assert.Empty(t, client.Phone)
	assert.Empty(t, client.Email)
	assert.Empty(t, client.Description)
}

func TestNewClient_WithEmptyName_ReturnsError(t *testing.T) {
	client, err := entities.NewClient(entities.CreateClientParams{
		UserID: uuid.New(),
	})

	assert.Nil(t, client)
	assert.ErrorIs(t, err, entities.ErrEmptyClientName)
}

func TestNewClient_WithNilUserID_ReturnsError(t *testing.T) {
	client, err := entities.NewClient(entities.CreateClientParams{
		Name: "Acme Corp",
	})

	assert.Nil(t, client)
	assert.ErrorIs(t, err, entities.ErrNilClientOwner)
}

func TestNewClient_WithInvalidPhone_ReturnsError(t *testing.T) {
	client, err := entities.NewClient(entities.CreateClientParams{
		Name:   "Acme Corp",
		Phone:  "not-a-phone",
		UserID: uuid.New(),
	})

	assert.Nil(t, client)
	assert.ErrorIs(t, err, valueobjects.ErrInvalidPhone)
}

func TestNewClient_WithInvalidEmail_ReturnsError(t *testing.T) {
	client, err := entities.NewClient(entities.CreateClientParams{
		Name:   "Acme Corp",
		Email:  "bad-email",
		UserID: uuid.New(),
	})

	assert.Nil(t, client)
	assert.ErrorIs(t, err, valueobjects.ErrInvalidEmail)
}

func TestNewClient_WithDescriptionTooLong_ReturnsError(t *testing.T) {
	tooLong := strings.Repeat("a", valueobjects.MaxDescriptionLength+1)
	client, err := entities.NewClient(entities.CreateClientParams{
		Name:        "Acme Corp",
		Description: tooLong,
		UserID:      uuid.New(),
	})

	assert.Nil(t, client)
	assert.ErrorIs(t, err, valueobjects.ErrDescriptionTooLong)
}

func TestNewClient_GeneratesUniqueIDs(t *testing.T) {
	userID := uuid.New()
	client1, err1 := entities.NewClient(entities.CreateClientParams{Name: "Client 1", UserID: userID})
	client2, err2 := entities.NewClient(entities.CreateClientParams{Name: "Client 2", UserID: userID})

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEqual(t, client1.ID, client2.ID, "each client must have a unique ID")
}

func TestNewClient_SetsCreatedAtAndUpdatedAtEqual(t *testing.T) {
	client, err := entities.NewClient(entities.CreateClientParams{Name: "Acme Corp", UserID: uuid.New()})

	require.NoError(t, err)
	assert.Equal(t, client.CreatedAt, client.UpdatedAt,
		"CreatedAt and UpdatedAt must be equal on creation")
}

func TestNewClient_StoresUserID(t *testing.T) {
	userID := uuid.New()
	client, err := entities.NewClient(entities.CreateClientParams{Name: "Acme Corp", UserID: userID})

	require.NoError(t, err)
	assert.Equal(t, userID, client.UserID, "client must store the owning user's ID")
}

// --- Setter tests ---

func newTestClientForSetters() *entities.Client {
	c, _ := entities.NewClient(entities.CreateClientParams{Name: "Acme", UserID: uuid.New()})
	return c
}

func TestSetPhone_WithValidPhone_Updates(t *testing.T) {
	client := newTestClientForSetters()
	err := client.SetPhone("+34 699 111 222")

	assert.NoError(t, err)
	assert.Equal(t, "+34 699 111 222", client.Phone)
}

func TestSetPhone_WithInvalidPhone_ReturnsError(t *testing.T) {
	client := newTestClientForSetters()
	err := client.SetPhone("invalid")

	assert.ErrorIs(t, err, valueobjects.ErrInvalidPhone)
}

func TestSetEmail_WithValidEmail_Updates(t *testing.T) {
	client := newTestClientForSetters()
	err := client.SetEmail("new@acme.com")

	assert.NoError(t, err)
	assert.Equal(t, "new@acme.com", client.Email)
}

func TestSetEmail_WithEmpty_ClearsEmail(t *testing.T) {
	client, _ := entities.NewClient(entities.CreateClientParams{
		Name:   "Acme",
		Email:  "old@acme.com",
		UserID: uuid.New(),
	})
	err := client.SetEmail("")

	assert.NoError(t, err)
	assert.Empty(t, client.Email)
}

func TestSetEmail_WithInvalidEmail_ReturnsError(t *testing.T) {
	client := newTestClientForSetters()
	err := client.SetEmail("bad")

	assert.ErrorIs(t, err, valueobjects.ErrInvalidEmail)
}

func TestSetDescription_WithValidDescription_Updates(t *testing.T) {
	client := newTestClientForSetters()
	err := client.SetDescription("Updated description")

	assert.NoError(t, err)
	assert.Equal(t, "Updated description", client.Description)
}

func TestSetDescription_TooLong_ReturnsError(t *testing.T) {
	client := newTestClientForSetters()
	err := client.SetDescription(strings.Repeat("x", valueobjects.MaxDescriptionLength+1))

	assert.ErrorIs(t, err, valueobjects.ErrDescriptionTooLong)
}
