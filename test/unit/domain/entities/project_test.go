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

func TestNewProject_WithValidData_ReturnsProject(t *testing.T) {
	clientID := uuid.New()
	project, err := entities.NewProject(entities.CreateProjectParams{
		Name:        "Residential Tower B",
		Address:     "Calle Mayor 12, Madrid",
		ManagerName: "Carlos Pérez",
		Phone:       "+34 699 111 222",
		Description: "14-storey residential building, phase 1",
		ClientID:    clientID,
	})

	require.NoError(t, err)
	assert.Equal(t, "Residential Tower B", project.Name)
	assert.Equal(t, "Calle Mayor 12, Madrid", project.Address)
	assert.Equal(t, "Carlos Pérez", project.ManagerName)
	assert.Equal(t, "+34 699 111 222", project.Phone)
	assert.Equal(t, "14-storey residential building, phase 1", project.Description)
	assert.Equal(t, clientID, project.ClientID)
	assert.NotEmpty(t, project.ID)
	assert.False(t, project.CreatedAt.IsZero())
	assert.False(t, project.UpdatedAt.IsZero())
	assert.Nil(t, project.DeletedAt)
}

func TestNewProject_WithOptionalFieldsEmpty_ReturnsProject(t *testing.T) {
	clientID := uuid.New()
	project, err := entities.NewProject(entities.CreateProjectParams{
		Name:     "Residential Tower B",
		ClientID: clientID,
	})

	require.NoError(t, err)
	assert.Equal(t, "Residential Tower B", project.Name)
	assert.Empty(t, project.Address)
	assert.Empty(t, project.ManagerName)
	assert.Empty(t, project.Phone)
	assert.Empty(t, project.Description)
}

func TestNewProject_WithEmptyName_ReturnsError(t *testing.T) {
	project, err := entities.NewProject(entities.CreateProjectParams{
		ClientID: uuid.New(),
	})

	assert.Nil(t, project)
	assert.ErrorIs(t, err, entities.ErrEmptyProjectName)
}

func TestNewProject_WithNilClientID_ReturnsError(t *testing.T) {
	project, err := entities.NewProject(entities.CreateProjectParams{
		Name: "Residential Tower B",
	})

	assert.Nil(t, project)
	assert.ErrorIs(t, err, entities.ErrNilProjectClient)
}

func TestNewProject_WithInvalidPhone_ReturnsError(t *testing.T) {
	project, err := entities.NewProject(entities.CreateProjectParams{
		Name:     "Residential Tower B",
		Phone:    "not-a-phone",
		ClientID: uuid.New(),
	})

	assert.Nil(t, project)
	assert.ErrorIs(t, err, valueobjects.ErrInvalidPhone)
}

func TestNewProject_WithDescriptionTooLong_ReturnsError(t *testing.T) {
	tooLong := strings.Repeat("a", valueobjects.MaxDescriptionLength+1)
	project, err := entities.NewProject(entities.CreateProjectParams{
		Name:        "Residential Tower B",
		Description: tooLong,
		ClientID:    uuid.New(),
	})

	assert.Nil(t, project)
	assert.ErrorIs(t, err, valueobjects.ErrDescriptionTooLong)
}

func TestNewProject_GeneratesUniqueIDs(t *testing.T) {
	clientID := uuid.New()
	p1, err1 := entities.NewProject(entities.CreateProjectParams{Name: "Project 1", ClientID: clientID})
	p2, err2 := entities.NewProject(entities.CreateProjectParams{Name: "Project 2", ClientID: clientID})

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEqual(t, p1.ID, p2.ID, "each project must have a unique ID")
}

func TestNewProject_SetsCreatedAtAndUpdatedAtEqual(t *testing.T) {
	project, err := entities.NewProject(entities.CreateProjectParams{
		Name:     "Residential Tower B",
		ClientID: uuid.New(),
	})

	require.NoError(t, err)
	assert.Equal(t, project.CreatedAt, project.UpdatedAt,
		"CreatedAt and UpdatedAt must be equal on creation")
}

func TestNewProject_StoresClientID(t *testing.T) {
	clientID := uuid.New()
	project, err := entities.NewProject(entities.CreateProjectParams{
		Name:     "Residential Tower B",
		ClientID: clientID,
	})

	require.NoError(t, err)
	assert.Equal(t, clientID, project.ClientID, "project must store the owning client's ID")
}

// --- UpdateProjectParams ---

func TestUpdateProjectParams_HasChanges_WithNoFields_ReturnsFalse(t *testing.T) {
	params := entities.UpdateProjectParams{}
	assert.False(t, params.HasChanges())
}

func TestUpdateProjectParams_HasChanges_WithAnyField_ReturnsTrue(t *testing.T) {
	name := "New Name"
	addr := "New Address"
	mgr := "New Manager"
	phone := "+34 600 000 000"
	desc := "New description"

	tests := []struct {
		label  string
		params entities.UpdateProjectParams
	}{
		{"Name", entities.UpdateProjectParams{Name: &name}},
		{"Address", entities.UpdateProjectParams{Address: &addr}},
		{"ManagerName", entities.UpdateProjectParams{ManagerName: &mgr}},
		{"Phone", entities.UpdateProjectParams{Phone: &phone}},
		{"Description", entities.UpdateProjectParams{Description: &desc}},
	}

	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			assert.True(t, tc.params.HasChanges())
		})
	}
}

// --- Setter tests ---

func newTestProjectForSetters() *entities.Project {
	p, _ := entities.NewProject(entities.CreateProjectParams{
		Name:     "Test Project",
		ClientID: uuid.New(),
	})
	return p
}

func TestSetName_WithValidName_Updates(t *testing.T) {
	project := newTestProjectForSetters()
	err := project.SetName("New Name")

	assert.NoError(t, err)
	assert.Equal(t, "New Name", project.Name)
}

func TestSetName_WithEmptyName_ReturnsError(t *testing.T) {
	project := newTestProjectForSetters()
	err := project.SetName("")

	assert.ErrorIs(t, err, entities.ErrEmptyProjectName)
}

func TestSetAddress_WithValidAddress_Updates(t *testing.T) {
	project := newTestProjectForSetters()
	err := project.SetAddress("Calle Mayor 12, Madrid")

	assert.NoError(t, err)
	assert.Equal(t, "Calle Mayor 12, Madrid", project.Address)
}

func TestSetAddress_WithEmpty_ClearsAddress(t *testing.T) {
	project, _ := entities.NewProject(entities.CreateProjectParams{
		Name:     "Test",
		Address:  "Old Address",
		ClientID: uuid.New(),
	})
	err := project.SetAddress("")

	assert.NoError(t, err)
	assert.Empty(t, project.Address)
}

func TestSetManagerName_WithValidName_Updates(t *testing.T) {
	project := newTestProjectForSetters()
	err := project.SetManagerName("Carlos Pérez")

	assert.NoError(t, err)
	assert.Equal(t, "Carlos Pérez", project.ManagerName)
}

func TestSetManagerName_WithEmpty_ClearsManagerName(t *testing.T) {
	project, _ := entities.NewProject(entities.CreateProjectParams{
		Name:        "Test",
		ManagerName: "Old Manager",
		ClientID:    uuid.New(),
	})
	err := project.SetManagerName("")

	assert.NoError(t, err)
	assert.Empty(t, project.ManagerName)
}

func TestProjectSetPhone_WithValidPhone_Updates(t *testing.T) {
	project := newTestProjectForSetters()
	err := project.SetPhone("+34 699 111 222")

	assert.NoError(t, err)
	assert.Equal(t, "+34 699 111 222", project.Phone)
}

func TestProjectSetPhone_WithEmpty_ClearsPhone(t *testing.T) {
	project, _ := entities.NewProject(entities.CreateProjectParams{
		Name:     "Test",
		Phone:    "+34 600 000 000",
		ClientID: uuid.New(),
	})
	err := project.SetPhone("")

	assert.NoError(t, err)
	assert.Empty(t, project.Phone)
}

func TestProjectSetPhone_WithInvalidPhone_ReturnsError(t *testing.T) {
	project := newTestProjectForSetters()
	err := project.SetPhone("invalid")

	assert.ErrorIs(t, err, valueobjects.ErrInvalidPhone)
}

func TestProjectSetDescription_WithValidDescription_Updates(t *testing.T) {
	project := newTestProjectForSetters()
	err := project.SetDescription("Updated description")

	assert.NoError(t, err)
	assert.Equal(t, "Updated description", project.Description)
}

func TestProjectSetDescription_WithEmpty_ClearsDescription(t *testing.T) {
	project, _ := entities.NewProject(entities.CreateProjectParams{
		Name:        "Test",
		Description: "Old description",
		ClientID:    uuid.New(),
	})
	err := project.SetDescription("")

	assert.NoError(t, err)
	assert.Empty(t, project.Description)
}

func TestProjectSetDescription_TooLong_ReturnsError(t *testing.T) {
	project := newTestProjectForSetters()
	err := project.SetDescription(strings.Repeat("x", valueobjects.MaxDescriptionLength+1))

	assert.ErrorIs(t, err, valueobjects.ErrDescriptionTooLong)
}
