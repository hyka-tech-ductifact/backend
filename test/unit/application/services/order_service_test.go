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
// CreateOrder
// =============================================================================

func TestCreateOrder_WithValidData_ReturnsOrder(t *testing.T) {
	userID := uuid.New()
	project := newTestProject(uuid.New())
	svc := services.NewOrderService(&mocks.MockOrderRepository{}, projectRepoReturning(project), &mocks.MockPieceRepository{})

	order, err := svc.CreateOrder(context.Background(), userID, entities.CreateOrderParams{
		Title:       "Steel beams – lot 3",
		Description: "First batch of structural steel",
		ProjectID:   project.ID,
	})

	require.NoError(t, err)
	assert.Equal(t, "Steel beams – lot 3", order.Title)
	assert.Equal(t, entities.OrderStatusPending, order.Status)
	assert.Equal(t, "First batch of structural steel", order.Description)
	assert.Equal(t, project.ID, order.ProjectID)
}

func TestCreateOrder_WithEmptyTitle_ReturnsError(t *testing.T) {
	userID := uuid.New()
	project := newTestProject(uuid.New())
	svc := services.NewOrderService(&mocks.MockOrderRepository{}, projectRepoReturning(project), &mocks.MockPieceRepository{})

	order, err := svc.CreateOrder(context.Background(), userID, entities.CreateOrderParams{
		ProjectID: project.ID,
	})

	assert.Nil(t, order)
	assert.ErrorIs(t, err, entities.ErrEmptyOrderTitle)
}

func TestCreateOrder_WithNonExistingProject_ReturnsError(t *testing.T) {
	svc := services.NewOrderService(&mocks.MockOrderRepository{}, projectRepoReturning(nil), &mocks.MockPieceRepository{})

	order, err := svc.CreateOrder(context.Background(), uuid.New(), entities.CreateOrderParams{
		Title:     "Steel beams",
		ProjectID: uuid.New(),
	})

	assert.Nil(t, order)
	assert.ErrorIs(t, err, repositories.ErrProjectNotFound)
}

func TestCreateOrder_WithProjectNotOwned_ReturnsError(t *testing.T) {
	projectRepo := &mocks.MockProjectRepository{
		GetByIDForOwnerFn: func(ctx context.Context, id uuid.UUID, ownerID uuid.UUID) (*entities.Project, error) {
			return nil, repositories.ErrProjectNotOwned
		},
	}
	svc := services.NewOrderService(&mocks.MockOrderRepository{}, projectRepo, &mocks.MockPieceRepository{})

	order, err := svc.CreateOrder(context.Background(), uuid.New(), entities.CreateOrderParams{
		Title:     "Steel beams",
		ProjectID: uuid.New(),
	})

	assert.Nil(t, order)
	assert.ErrorIs(t, err, repositories.ErrProjectNotOwned)
}

func TestCreateOrder_WhenRepoFails_ReturnsError(t *testing.T) {
	userID := uuid.New()
	project := newTestProject(uuid.New())
	orderRepo := &mocks.MockOrderRepository{
		CreateFn: func(ctx context.Context, order *entities.Order) error {
			return errors.New("db connection lost")
		},
	}
	svc := services.NewOrderService(orderRepo, projectRepoReturning(project), &mocks.MockPieceRepository{})

	order, err := svc.CreateOrder(context.Background(), userID, entities.CreateOrderParams{
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
	order := newTestOrder(uuid.New())
	svc := services.NewOrderService(orderRepoReturning(order), &mocks.MockProjectRepository{}, &mocks.MockPieceRepository{})

	result, err := svc.GetOrderByID(context.Background(), order.ID, userID)

	require.NoError(t, err)
	assert.Equal(t, order.Title, result.Title)
	assert.Equal(t, order.ID, result.ID)
}

func TestGetOrderByID_WithNonExistingOrder_ReturnsError(t *testing.T) {
	svc := services.NewOrderService(orderRepoReturning(nil), &mocks.MockProjectRepository{}, &mocks.MockPieceRepository{})

	result, err := svc.GetOrderByID(context.Background(), uuid.New(), uuid.New())

	assert.Nil(t, result)
	assert.ErrorIs(t, err, repositories.ErrOrderNotFound)
}

func TestGetOrderByID_WithNotOwned_ReturnsError(t *testing.T) {
	orderRepo := &mocks.MockOrderRepository{
		GetByIDForOwnerFn: func(ctx context.Context, id uuid.UUID, ownerID uuid.UUID) (*entities.Order, error) {
			return nil, repositories.ErrOrderNotOwned
		},
	}
	svc := services.NewOrderService(orderRepo, &mocks.MockProjectRepository{}, &mocks.MockPieceRepository{})

	result, err := svc.GetOrderByID(context.Background(), uuid.New(), uuid.New())

	assert.Nil(t, result)
	assert.ErrorIs(t, err, repositories.ErrOrderNotOwned)
}

// =============================================================================
// ListOrdersByProjectID
// =============================================================================

func TestListOrdersByProjectID_ReturnsOrders(t *testing.T) {
	userID := uuid.New()
	project := newTestProject(uuid.New())
	expected := []*entities.Order{
		{ID: uuid.New(), Title: "Order 1", ProjectID: project.ID},
		{ID: uuid.New(), Title: "Order 2", ProjectID: project.ID},
	}
	orderRepo := &mocks.MockOrderRepository{
		ListByProjectIDFn: func(ctx context.Context, pID uuid.UUID, pg pagination.Pagination) ([]*entities.Order, int64, error) {
			return expected, 2, nil
		},
	}
	svc := services.NewOrderService(orderRepo, projectRepoReturning(project), &mocks.MockPieceRepository{})

	pg, _ := pagination.NewPagination(1, 20)
	result, err := svc.ListOrdersByProjectID(context.Background(), project.ID, userID, pg)

	require.NoError(t, err)
	assert.Len(t, result.Data, 2)
	assert.Equal(t, int64(2), result.TotalItems)
}

func TestListOrdersByProjectID_EmptyList(t *testing.T) {
	userID := uuid.New()
	project := newTestProject(uuid.New())
	orderRepo := &mocks.MockOrderRepository{
		ListByProjectIDFn: func(ctx context.Context, pID uuid.UUID, pg pagination.Pagination) ([]*entities.Order, int64, error) {
			return []*entities.Order{}, 0, nil
		},
	}
	svc := services.NewOrderService(orderRepo, projectRepoReturning(project), &mocks.MockPieceRepository{})

	pg, _ := pagination.NewPagination(1, 20)
	result, err := svc.ListOrdersByProjectID(context.Background(), project.ID, userID, pg)

	require.NoError(t, err)
	assert.Empty(t, result.Data)
	assert.Equal(t, int64(0), result.TotalItems)
}

func TestListOrdersByProjectID_WithProjectNotOwned_ReturnsError(t *testing.T) {
	projectRepo := &mocks.MockProjectRepository{
		GetByIDForOwnerFn: func(ctx context.Context, id uuid.UUID, ownerID uuid.UUID) (*entities.Project, error) {
			return nil, repositories.ErrProjectNotOwned
		},
	}
	svc := services.NewOrderService(&mocks.MockOrderRepository{}, projectRepo, &mocks.MockPieceRepository{})

	pg, _ := pagination.NewPagination(1, 20)
	_, err := svc.ListOrdersByProjectID(context.Background(), uuid.New(), uuid.New(), pg)

	assert.ErrorIs(t, err, repositories.ErrProjectNotOwned)
}

// =============================================================================
// UpdateOrder
// =============================================================================

func TestUpdateOrder_WithNewTitle_UpdatesTitle(t *testing.T) {
	userID := uuid.New()
	order := newTestOrder(uuid.New())
	svc := services.NewOrderService(orderRepoReturning(order), &mocks.MockProjectRepository{}, &mocks.MockPieceRepository{})

	result, err := svc.UpdateOrder(context.Background(), order.ID, userID, entities.UpdateOrderParams{
		Title: strPtr("New Title"),
	})

	require.NoError(t, err)
	assert.Equal(t, "New Title", result.Title)
}

func TestUpdateOrder_WithStatus_UpdatesStatus(t *testing.T) {
	userID := uuid.New()
	order := newTestOrder(uuid.New())
	svc := services.NewOrderService(orderRepoReturning(order), &mocks.MockProjectRepository{}, &mocks.MockPieceRepository{})

	completed := "completed"
	result, err := svc.UpdateOrder(context.Background(), order.ID, userID, entities.UpdateOrderParams{
		Status: &completed,
	})

	require.NoError(t, err)
	assert.Equal(t, entities.OrderStatusCompleted, result.Status)
}

func TestUpdateOrder_WithDescription_UpdatesDescription(t *testing.T) {
	userID := uuid.New()
	order := newTestOrder(uuid.New())
	svc := services.NewOrderService(orderRepoReturning(order), &mocks.MockProjectRepository{}, &mocks.MockPieceRepository{})

	result, err := svc.UpdateOrder(context.Background(), order.ID, userID, entities.UpdateOrderParams{
		Description: strPtr("Updated description"),
	})

	require.NoError(t, err)
	assert.Equal(t, "Updated description", result.Description)
}

func TestUpdateOrder_WithEmptyTitle_ReturnsError(t *testing.T) {
	userID := uuid.New()
	order := newTestOrder(uuid.New())
	svc := services.NewOrderService(orderRepoReturning(order), &mocks.MockProjectRepository{}, &mocks.MockPieceRepository{})

	result, err := svc.UpdateOrder(context.Background(), order.ID, userID, entities.UpdateOrderParams{
		Title: strPtr(""),
	})

	assert.Nil(t, result)
	assert.ErrorIs(t, err, entities.ErrEmptyOrderTitle)
}

func TestUpdateOrder_WithInvalidStatus_ReturnsError(t *testing.T) {
	userID := uuid.New()
	order := newTestOrder(uuid.New())
	svc := services.NewOrderService(orderRepoReturning(order), &mocks.MockProjectRepository{}, &mocks.MockPieceRepository{})

	invalid := "invalid"
	result, err := svc.UpdateOrder(context.Background(), order.ID, userID, entities.UpdateOrderParams{
		Status: &invalid,
	})

	assert.Nil(t, result)
	assert.ErrorIs(t, err, entities.ErrInvalidOrderStatus)
}

func TestUpdateOrder_WithNoChanges_SkipsPersistence(t *testing.T) {
	userID := uuid.New()
	order := newTestOrder(uuid.New())
	repo := orderRepoReturning(order)
	updateCalled := false
	repo.UpdateFn = func(ctx context.Context, o *entities.Order) error {
		updateCalled = true
		return nil
	}
	svc := services.NewOrderService(repo, &mocks.MockProjectRepository{}, &mocks.MockPieceRepository{})

	_, err := svc.UpdateOrder(context.Background(), order.ID, userID, entities.UpdateOrderParams{})

	require.NoError(t, err)
	assert.False(t, updateCalled, "Update should not be called when no fields change")
}

func TestUpdateOrder_UpdatesTimestamp(t *testing.T) {
	userID := uuid.New()
	order := newTestOrder(uuid.New())
	oldTime := order.UpdatedAt
	svc := services.NewOrderService(orderRepoReturning(order), &mocks.MockProjectRepository{}, &mocks.MockPieceRepository{})

	result, err := svc.UpdateOrder(context.Background(), order.ID, userID, entities.UpdateOrderParams{
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
	order := newTestOrder(uuid.New())
	deleteCalled := false
	repo := orderRepoReturning(order)
	repo.DeleteFn = func(ctx context.Context, id uuid.UUID) error {
		deleteCalled = true
		return nil
	}
	svc := services.NewOrderService(repo, &mocks.MockProjectRepository{}, &mocks.MockPieceRepository{})

	err := svc.DeleteOrder(context.Background(), order.ID, userID, true)

	assert.NoError(t, err)
	assert.True(t, deleteCalled)
}

func TestDeleteOrder_WithNonExistingOrder_ReturnsError(t *testing.T) {
	svc := services.NewOrderService(orderRepoReturning(nil), &mocks.MockProjectRepository{}, &mocks.MockPieceRepository{})

	err := svc.DeleteOrder(context.Background(), uuid.New(), uuid.New(), true)

	assert.ErrorIs(t, err, repositories.ErrOrderNotFound)
}

func TestDeleteOrder_WithNotOwned_ReturnsError(t *testing.T) {
	orderRepo := &mocks.MockOrderRepository{
		GetByIDForOwnerFn: func(ctx context.Context, id uuid.UUID, ownerID uuid.UUID) (*entities.Order, error) {
			return nil, repositories.ErrOrderNotOwned
		},
	}
	svc := services.NewOrderService(orderRepo, &mocks.MockProjectRepository{}, &mocks.MockPieceRepository{})

	err := svc.DeleteOrder(context.Background(), uuid.New(), uuid.New(), true)

	assert.ErrorIs(t, err, repositories.ErrOrderNotOwned)
}

func TestDeleteOrder_WithPieces_NoCascade_ReturnsConflict(t *testing.T) {
	userID := uuid.New()
	order := newTestOrder(uuid.New())
	pieceRepo := &mocks.MockPieceRepository{
		CountByOrderIDFn: func(ctx context.Context, orderID uuid.UUID) (int64, error) {
			return 10, nil
		},
	}
	svc := services.NewOrderService(orderRepoReturning(order), &mocks.MockProjectRepository{}, pieceRepo)

	err := svc.DeleteOrder(context.Background(), order.ID, userID, false)

	assert.ErrorIs(t, err, services.ErrHasAssociatedPieces)
}

func TestDeleteOrder_WithPieces_Cascade_Succeeds(t *testing.T) {
	userID := uuid.New()
	order := newTestOrder(uuid.New())
	deleteCalled := false
	repo := orderRepoReturning(order)
	repo.DeleteFn = func(ctx context.Context, id uuid.UUID) error {
		deleteCalled = true
		return nil
	}
	pieceRepo := &mocks.MockPieceRepository{
		CountByOrderIDFn: func(ctx context.Context, orderID uuid.UUID) (int64, error) {
			return 10, nil
		},
	}
	svc := services.NewOrderService(repo, &mocks.MockProjectRepository{}, pieceRepo)

	err := svc.DeleteOrder(context.Background(), order.ID, userID, true)

	assert.NoError(t, err)
	assert.True(t, deleteCalled)
}
