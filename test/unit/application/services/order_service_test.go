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
// CreateOrder
// =============================================================================

func TestCreateOrder_WithValidData_ReturnsOrder(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	project := newTestProject(client.ID)
	svc := services.NewOrderService(&mocks.MockOrderRepository{}, projectRepoReturning(project), clientRepoReturning(client))

	order, err := svc.CreateOrder(context.Background(), userID, client.ID, entities.CreateOrderParams{
		Title:     "Steel beams – lot 3",
		ProjectID: project.ID,
	})

	require.NoError(t, err)
	assert.Equal(t, "Steel beams – lot 3", order.Title)
	assert.Equal(t, entities.OrderStatusPending, order.Status)
	assert.Equal(t, project.ID, order.ProjectID)
}

func TestCreateOrder_WithEmptyTitle_ReturnsError(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	project := newTestProject(client.ID)
	svc := services.NewOrderService(&mocks.MockOrderRepository{}, projectRepoReturning(project), clientRepoReturning(client))

	order, err := svc.CreateOrder(context.Background(), userID, client.ID, entities.CreateOrderParams{
		ProjectID: project.ID,
	})

	assert.Nil(t, order)
	assert.ErrorIs(t, err, entities.ErrEmptyOrderTitle)
}

func TestCreateOrder_WithNonExistingClient_ReturnsError(t *testing.T) {
	svc := services.NewOrderService(&mocks.MockOrderRepository{}, &mocks.MockProjectRepository{}, clientRepoReturning(nil))

	order, err := svc.CreateOrder(context.Background(), uuid.New(), uuid.New(), entities.CreateOrderParams{
		Title:     "Steel beams",
		ProjectID: uuid.New(),
	})

	assert.Nil(t, order)
	assert.ErrorIs(t, err, services.ErrClientNotFound)
}

func TestCreateOrder_WithWrongUser_ReturnsError(t *testing.T) {
	ownerID := uuid.New()
	client := newTestClient(ownerID)
	project := newTestProject(client.ID)
	svc := services.NewOrderService(&mocks.MockOrderRepository{}, projectRepoReturning(project), clientRepoReturning(client))

	order, err := svc.CreateOrder(context.Background(), uuid.New(), client.ID, entities.CreateOrderParams{
		Title:     "Steel beams",
		ProjectID: project.ID,
	})

	assert.Nil(t, order)
	assert.ErrorIs(t, err, services.ErrClientNotOwned)
}

func TestCreateOrder_WithNonExistingProject_ReturnsError(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	svc := services.NewOrderService(&mocks.MockOrderRepository{}, projectRepoReturning(nil), clientRepoReturning(client))

	order, err := svc.CreateOrder(context.Background(), userID, client.ID, entities.CreateOrderParams{
		Title:     "Steel beams",
		ProjectID: uuid.New(),
	})

	assert.Nil(t, order)
	assert.ErrorIs(t, err, services.ErrProjectNotFound)
}

func TestCreateOrder_WithWrongClient_ReturnsError(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	otherClientID := uuid.New()
	project := newTestProject(otherClientID) // project belongs to a different client
	svc := services.NewOrderService(&mocks.MockOrderRepository{}, projectRepoReturning(project), clientRepoReturning(client))

	order, err := svc.CreateOrder(context.Background(), userID, client.ID, entities.CreateOrderParams{
		Title:     "Steel beams",
		ProjectID: project.ID,
	})

	assert.Nil(t, order)
	assert.ErrorIs(t, err, services.ErrProjectNotOwned)
}

func TestCreateOrder_WhenRepoFails_ReturnsError(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	project := newTestProject(client.ID)
	orderRepo := &mocks.MockOrderRepository{
		CreateFn: func(ctx context.Context, order *entities.Order) error {
			return errors.New("db connection lost")
		},
	}
	svc := services.NewOrderService(orderRepo, projectRepoReturning(project), clientRepoReturning(client))

	order, err := svc.CreateOrder(context.Background(), userID, client.ID, entities.CreateOrderParams{
		Title:     "Steel beams",
		ProjectID: project.ID,
	})

	assert.Nil(t, order)
	assert.EqualError(t, err, "db connection lost")
}

// =============================================================================
// GetOrderByID
// =============================================================================

func TestGetOrderByID_WithExistingOrder_ReturnsOrder(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	project := newTestProject(client.ID)
	order := newTestOrder(project.ID)
	svc := services.NewOrderService(orderRepoReturning(order), projectRepoReturning(project), clientRepoReturning(client))

	result, err := svc.GetOrderByID(context.Background(), order.ID, project.ID, client.ID, userID)

	require.NoError(t, err)
	assert.Equal(t, order.Title, result.Title)
	assert.Equal(t, order.ID, result.ID)
}

func TestGetOrderByID_WithNonExistingOrder_ReturnsError(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	project := newTestProject(client.ID)
	svc := services.NewOrderService(orderRepoReturning(nil), projectRepoReturning(project), clientRepoReturning(client))

	result, err := svc.GetOrderByID(context.Background(), uuid.New(), project.ID, client.ID, userID)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, services.ErrOrderNotFound)
}

func TestGetOrderByID_WithWrongProject_ReturnsError(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	project := newTestProject(client.ID)
	otherProjectID := uuid.New()
	order := newTestOrder(otherProjectID) // order belongs to a different project
	svc := services.NewOrderService(orderRepoReturning(order), projectRepoReturning(project), clientRepoReturning(client))

	result, err := svc.GetOrderByID(context.Background(), order.ID, project.ID, client.ID, userID)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, services.ErrOrderNotOwned)
}

// =============================================================================
// ListOrdersByProjectID
// =============================================================================

func TestListOrdersByProjectID_ReturnsOrders(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	project := newTestProject(client.ID)
	expected := []*entities.Order{
		{ID: uuid.New(), Title: "Order 1", ProjectID: project.ID},
		{ID: uuid.New(), Title: "Order 2", ProjectID: project.ID},
	}
	orderRepo := &mocks.MockOrderRepository{
		ListByProjectIDFn: func(ctx context.Context, pID uuid.UUID, pg pagination.Pagination) ([]*entities.Order, int64, error) {
			return expected, 2, nil
		},
	}
	svc := services.NewOrderService(orderRepo, projectRepoReturning(project), clientRepoReturning(client))

	pg, _ := pagination.NewPagination(1, 20)
	result, err := svc.ListOrdersByProjectID(context.Background(), project.ID, client.ID, userID, pg)

	require.NoError(t, err)
	assert.Len(t, result.Data, 2)
	assert.Equal(t, int64(2), result.TotalItems)
}

func TestListOrdersByProjectID_EmptyList(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	project := newTestProject(client.ID)
	orderRepo := &mocks.MockOrderRepository{
		ListByProjectIDFn: func(ctx context.Context, pID uuid.UUID, pg pagination.Pagination) ([]*entities.Order, int64, error) {
			return []*entities.Order{}, 0, nil
		},
	}
	svc := services.NewOrderService(orderRepo, projectRepoReturning(project), clientRepoReturning(client))

	pg, _ := pagination.NewPagination(1, 20)
	result, err := svc.ListOrdersByProjectID(context.Background(), project.ID, client.ID, userID, pg)

	require.NoError(t, err)
	assert.Empty(t, result.Data)
	assert.Equal(t, int64(0), result.TotalItems)
}

func TestListOrdersByProjectID_WithWrongUser_ReturnsError(t *testing.T) {
	ownerID := uuid.New()
	client := newTestClient(ownerID)
	project := newTestProject(client.ID)
	svc := services.NewOrderService(&mocks.MockOrderRepository{}, projectRepoReturning(project), clientRepoReturning(client))

	pg, _ := pagination.NewPagination(1, 20)
	_, err := svc.ListOrdersByProjectID(context.Background(), project.ID, client.ID, uuid.New(), pg)

	assert.ErrorIs(t, err, services.ErrClientNotOwned)
}

// =============================================================================
// UpdateOrder
// =============================================================================

func TestUpdateOrder_WithNewTitle_UpdatesTitle(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	project := newTestProject(client.ID)
	order := newTestOrder(project.ID)
	svc := services.NewOrderService(orderRepoReturning(order), projectRepoReturning(project), clientRepoReturning(client))

	result, err := svc.UpdateOrder(context.Background(), order.ID, project.ID, client.ID, userID, entities.UpdateOrderParams{
		Title: strPtr("New Title"),
	})

	require.NoError(t, err)
	assert.Equal(t, "New Title", result.Title)
}

func TestUpdateOrder_WithStatus_UpdatesStatus(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	project := newTestProject(client.ID)
	order := newTestOrder(project.ID)
	svc := services.NewOrderService(orderRepoReturning(order), projectRepoReturning(project), clientRepoReturning(client))

	completed := "completed"
	result, err := svc.UpdateOrder(context.Background(), order.ID, project.ID, client.ID, userID, entities.UpdateOrderParams{
		Status: &completed,
	})

	require.NoError(t, err)
	assert.Equal(t, entities.OrderStatusCompleted, result.Status)
}

func TestUpdateOrder_WithEmptyTitle_ReturnsError(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	project := newTestProject(client.ID)
	order := newTestOrder(project.ID)
	svc := services.NewOrderService(orderRepoReturning(order), projectRepoReturning(project), clientRepoReturning(client))

	result, err := svc.UpdateOrder(context.Background(), order.ID, project.ID, client.ID, userID, entities.UpdateOrderParams{
		Title: strPtr(""),
	})

	assert.Nil(t, result)
	assert.ErrorIs(t, err, entities.ErrEmptyOrderTitle)
}

func TestUpdateOrder_WithInvalidStatus_ReturnsError(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	project := newTestProject(client.ID)
	order := newTestOrder(project.ID)
	svc := services.NewOrderService(orderRepoReturning(order), projectRepoReturning(project), clientRepoReturning(client))

	invalid := "invalid"
	result, err := svc.UpdateOrder(context.Background(), order.ID, project.ID, client.ID, userID, entities.UpdateOrderParams{
		Status: &invalid,
	})

	assert.Nil(t, result)
	assert.ErrorIs(t, err, entities.ErrInvalidOrderStatus)
}

func TestUpdateOrder_WithNoChanges_SkipsPersistence(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	project := newTestProject(client.ID)
	order := newTestOrder(project.ID)
	repo := orderRepoReturning(order)
	updateCalled := false
	repo.UpdateFn = func(ctx context.Context, o *entities.Order) error {
		updateCalled = true
		return nil
	}
	svc := services.NewOrderService(repo, projectRepoReturning(project), clientRepoReturning(client))

	_, err := svc.UpdateOrder(context.Background(), order.ID, project.ID, client.ID, userID, entities.UpdateOrderParams{})

	require.NoError(t, err)
	assert.False(t, updateCalled, "Update should not be called when no fields change")
}

func TestUpdateOrder_UpdatesTimestamp(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	project := newTestProject(client.ID)
	order := newTestOrder(project.ID)
	oldTime := order.UpdatedAt
	svc := services.NewOrderService(orderRepoReturning(order), projectRepoReturning(project), clientRepoReturning(client))

	result, err := svc.UpdateOrder(context.Background(), order.ID, project.ID, client.ID, userID, entities.UpdateOrderParams{
		Title: strPtr("New Title"),
	})

	require.NoError(t, err)
	assert.True(t, result.UpdatedAt.After(oldTime), "UpdatedAt must be newer than the original")
}

// =============================================================================
// DeleteOrder
// =============================================================================

func TestDeleteOrder_WithExistingOrder_Succeeds(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	project := newTestProject(client.ID)
	order := newTestOrder(project.ID)
	deleteCalled := false
	repo := orderRepoReturning(order)
	repo.DeleteFn = func(ctx context.Context, id uuid.UUID) error {
		deleteCalled = true
		return nil
	}
	svc := services.NewOrderService(repo, projectRepoReturning(project), clientRepoReturning(client))

	err := svc.DeleteOrder(context.Background(), order.ID, project.ID, client.ID, userID)

	assert.NoError(t, err)
	assert.True(t, deleteCalled)
}

func TestDeleteOrder_WithNonExistingOrder_ReturnsError(t *testing.T) {
	userID := uuid.New()
	client := newTestClient(userID)
	project := newTestProject(client.ID)
	svc := services.NewOrderService(orderRepoReturning(nil), projectRepoReturning(project), clientRepoReturning(client))

	err := svc.DeleteOrder(context.Background(), uuid.New(), project.ID, client.ID, userID)

	assert.ErrorIs(t, err, services.ErrOrderNotFound)
}

func TestDeleteOrder_WithWrongUser_ReturnsError(t *testing.T) {
	ownerID := uuid.New()
	client := newTestClient(ownerID)
	project := newTestProject(client.ID)
	order := newTestOrder(project.ID)
	svc := services.NewOrderService(orderRepoReturning(order), projectRepoReturning(project), clientRepoReturning(client))

	err := svc.DeleteOrder(context.Background(), order.ID, project.ID, client.ID, uuid.New())

	assert.ErrorIs(t, err, services.ErrClientNotOwned)
}
