package entities_test

import (
	"testing"

	"ductifact/internal/domain/entities"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOrder_WithValidData_ReturnsOrder(t *testing.T) {
	projectID := uuid.New()
	order, err := entities.NewOrder(entities.CreateOrderParams{
		Title:       "Steel beams – lot 3",
		Status:      "pending",
		Description: "First batch of structural steel",
		ProjectID:   projectID,
	})

	require.NoError(t, err)
	assert.Equal(t, "Steel beams – lot 3", order.Title)
	assert.Equal(t, entities.OrderStatusPending, order.Status)
	assert.Equal(t, "First batch of structural steel", order.Description)
	assert.Equal(t, projectID, order.ProjectID)
	assert.NotEmpty(t, order.ID)
	assert.False(t, order.CreatedAt.IsZero())
	assert.False(t, order.UpdatedAt.IsZero())
	assert.Nil(t, order.DeletedAt)
}

func TestNewOrder_WithDefaultStatus_ReturnsPending(t *testing.T) {
	order, err := entities.NewOrder(entities.CreateOrderParams{
		Title:     "Steel beams – lot 3",
		ProjectID: uuid.New(),
	})

	require.NoError(t, err)
	assert.Equal(t, entities.OrderStatusPending, order.Status)
	assert.Equal(t, "", order.Description, "description defaults to empty string")
}

func TestNewOrder_WithCompletedStatus_ReturnsCompleted(t *testing.T) {
	order, err := entities.NewOrder(entities.CreateOrderParams{
		Title:     "Steel beams – lot 3",
		Status:    "completed",
		ProjectID: uuid.New(),
	})

	require.NoError(t, err)
	assert.Equal(t, entities.OrderStatusCompleted, order.Status)
}

func TestNewOrder_WithEmptyTitle_ReturnsError(t *testing.T) {
	order, err := entities.NewOrder(entities.CreateOrderParams{
		ProjectID: uuid.New(),
	})

	assert.Nil(t, order)
	assert.ErrorIs(t, err, entities.ErrEmptyOrderTitle)
}

func TestNewOrder_WithNilProjectID_ReturnsError(t *testing.T) {
	order, err := entities.NewOrder(entities.CreateOrderParams{
		Title: "Steel beams – lot 3",
	})

	assert.Nil(t, order)
	assert.ErrorIs(t, err, entities.ErrNilOrderProject)
}

func TestNewOrder_WithInvalidStatus_ReturnsError(t *testing.T) {
	order, err := entities.NewOrder(entities.CreateOrderParams{
		Title:     "Steel beams – lot 3",
		Status:    "invalid",
		ProjectID: uuid.New(),
	})

	assert.Nil(t, order)
	assert.ErrorIs(t, err, entities.ErrInvalidOrderStatus)
}

func TestNewOrder_GeneratesUniqueIDs(t *testing.T) {
	projectID := uuid.New()
	o1, err1 := entities.NewOrder(entities.CreateOrderParams{Title: "Order 1", ProjectID: projectID})
	o2, err2 := entities.NewOrder(entities.CreateOrderParams{Title: "Order 2", ProjectID: projectID})

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEqual(t, o1.ID, o2.ID, "each order must have a unique ID")
}

func TestNewOrder_SetsCreatedAtAndUpdatedAtEqual(t *testing.T) {
	order, err := entities.NewOrder(entities.CreateOrderParams{
		Title:     "Steel beams",
		ProjectID: uuid.New(),
	})

	require.NoError(t, err)
	assert.Equal(t, order.CreatedAt, order.UpdatedAt,
		"CreatedAt and UpdatedAt must be equal on creation")
}

func TestNewOrder_StoresProjectID(t *testing.T) {
	projectID := uuid.New()
	order, err := entities.NewOrder(entities.CreateOrderParams{
		Title:     "Steel beams",
		ProjectID: projectID,
	})

	require.NoError(t, err)
	assert.Equal(t, projectID, order.ProjectID, "order must store the owning project's ID")
}

// --- UpdateOrderParams ---

func TestUpdateOrderParams_HasChanges_WithNoFields_ReturnsFalse(t *testing.T) {
	params := entities.UpdateOrderParams{}
	assert.False(t, params.HasChanges())
}

func TestUpdateOrderParams_HasChanges_WithAnyField_ReturnsTrue(t *testing.T) {
	title := "New Title"
	status := "completed"
	desc := "Updated description"

	tests := []struct {
		label  string
		params entities.UpdateOrderParams
	}{
		{"Title", entities.UpdateOrderParams{Title: &title}},
		{"Status", entities.UpdateOrderParams{Status: &status}},
		{"Description", entities.UpdateOrderParams{Description: &desc}},
	}

	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			assert.True(t, tc.params.HasChanges())
		})
	}
}

// --- Setter tests ---

func newTestOrderForSetters() *entities.Order {
	o, _ := entities.NewOrder(entities.CreateOrderParams{
		Title:     "Test Order",
		ProjectID: uuid.New(),
	})
	return o
}

func TestOrderSetTitle_WithValidTitle_Updates(t *testing.T) {
	order := newTestOrderForSetters()
	err := order.SetTitle("New Title")

	assert.NoError(t, err)
	assert.Equal(t, "New Title", order.Title)
}

func TestOrderSetTitle_WithEmptyTitle_ReturnsError(t *testing.T) {
	order := newTestOrderForSetters()
	err := order.SetTitle("")

	assert.ErrorIs(t, err, entities.ErrEmptyOrderTitle)
}

func TestOrderSetStatus_WithValidStatus_Updates(t *testing.T) {
	order := newTestOrderForSetters()
	err := order.SetStatus("completed")

	assert.NoError(t, err)
	assert.Equal(t, entities.OrderStatusCompleted, order.Status)
}

func TestOrderSetStatus_WithInvalidStatus_ReturnsError(t *testing.T) {
	order := newTestOrderForSetters()
	err := order.SetStatus("invalid")

	assert.ErrorIs(t, err, entities.ErrInvalidOrderStatus)
}

func TestOrderSetDescription_Updates(t *testing.T) {
	order := newTestOrderForSetters()
	order.SetDescription("New description")

	assert.Equal(t, "New description", order.Description)
}

func TestOrderSetDescription_WithEmpty_ClearsField(t *testing.T) {
	order := newTestOrderForSetters()
	order.SetDescription("some text")
	order.SetDescription("")

	assert.Equal(t, "", order.Description)
}

// --- OrderStatus.IsValid ---

func TestOrderStatus_IsValid(t *testing.T) {
	assert.True(t, entities.OrderStatusPending.IsValid())
	assert.True(t, entities.OrderStatusCompleted.IsValid())
	assert.False(t, entities.OrderStatus("unknown").IsValid())
}
