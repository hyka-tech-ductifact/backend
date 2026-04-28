package persistence

import (
	"context"
	"errors"
	"time"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/pagination"
	"ductifact/internal/domain/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// --- Database Model (infrastructure concern) ---

// OrderModel is the GORM-specific database representation.
// It is NOT a domain entity. It lives here because only this adapter cares about it.
type OrderModel struct {
	ID          uuid.UUID `gorm:"primaryKey"`
	Title       string
	Status      string
	Description string
	ProjectID   uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt

	// GORM association to ProjectModel for FK constraint and cascade delete.
	Project ProjectModel `gorm:"foreignKey:ProjectID"`
}

func (OrderModel) TableName() string {
	return "orders"
}

// --- Repository implementation ---

// PostgresOrderRepository implements domain's OrderRepository interface.
type PostgresOrderRepository struct {
	db *gorm.DB
}

func NewPostgresOrderRepository(db *gorm.DB) *PostgresOrderRepository {
	return &PostgresOrderRepository{db: db}
}

func (r *PostgresOrderRepository) Create(ctx context.Context, order *entities.Order) error {
	model := toOrderModel(order)
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *PostgresOrderRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Order, error) {
	var model OrderModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}
	return toOrderEntity(&model), nil
}

func (r *PostgresOrderRepository) GetByIDForOwner(ctx context.Context, id uuid.UUID, ownerID uuid.UUID) (*entities.Order, error) {
	var model OrderModel
	err := r.db.WithContext(ctx).
		Joins("JOIN projects ON projects.id = orders.project_id AND projects.deleted_at IS NULL").
		Joins("JOIN clients ON clients.id = projects.client_id AND clients.deleted_at IS NULL").
		Where("orders.id = ? AND clients.user_id = ?", id, ownerID).
		First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, r.diagnoseOrderFailure(ctx, id, ownerID)
	}
	if err != nil {
		return nil, err
	}
	return toOrderEntity(&model), nil
}

func (r *PostgresOrderRepository) diagnoseOrderFailure(ctx context.Context, id uuid.UUID, ownerID uuid.UUID) error {
	var order OrderModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return repositories.ErrOrderNotFound
		}
		return err
	}
	var project ProjectModel
	if err := r.db.WithContext(ctx).Where("id = ?", order.ProjectID).First(&project).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return repositories.ErrProjectNotFound
		}
		return err
	}
	var client ClientModel
	if err := r.db.WithContext(ctx).Where("id = ?", project.ClientID).First(&client).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return repositories.ErrClientNotFound
		}
		return err
	}
	if client.UserID != ownerID {
		return repositories.ErrOrderNotOwned
	}
	return repositories.ErrOrderNotFound
}

func (r *PostgresOrderRepository) ListByProjectID(ctx context.Context, projectID uuid.UUID, pg pagination.Pagination) ([]*entities.Order, int64, error) {
	var totalItems int64

	// Count total matching rows (before pagination)
	if err := r.db.WithContext(ctx).Model(&OrderModel{}).Where("project_id = ?", projectID).Count(&totalItems).Error; err != nil {
		return nil, 0, err
	}

	// Fetch the requested page
	var models []OrderModel
	err := r.db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Order("created_at DESC").
		Offset((pg.Page - 1) * pg.PageSize).
		Limit(pg.PageSize).
		Find(&models).Error
	if err != nil {
		return nil, 0, err
	}

	orders := make([]*entities.Order, len(models))
	for i := range models {
		orders[i] = toOrderEntity(&models[i])
	}
	return orders, totalItems, nil
}

func (r *PostgresOrderRepository) Update(ctx context.Context, order *entities.Order) error {
	order.UpdatedAt = time.Now()
	model := toOrderModel(order)
	return r.db.WithContext(ctx).Save(model).Error
}

func (r *PostgresOrderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&OrderModel{}, "id = ?", id).Error
}

func (r *PostgresOrderRepository) CountByProjectID(ctx context.Context, projectID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&OrderModel{}).Where("project_id = ?", projectID).Count(&count).Error
	return count, err
}

// --- Mappers (package-level functions, not methods) ---

func toOrderModel(order *entities.Order) *OrderModel {
	model := &OrderModel{
		ID:          order.ID,
		Title:       order.Title,
		Status:      string(order.Status),
		Description: order.Description,
		ProjectID:   order.ProjectID,
		CreatedAt:   order.CreatedAt,
		UpdatedAt:   order.UpdatedAt,
	}
	if order.DeletedAt != nil {
		model.DeletedAt = gorm.DeletedAt{Time: *order.DeletedAt, Valid: true}
	}
	return model
}

func toOrderEntity(model *OrderModel) *entities.Order {
	entity := &entities.Order{
		ID:          model.ID,
		Title:       model.Title,
		Status:      entities.OrderStatus(model.Status),
		Description: model.Description,
		ProjectID:   model.ProjectID,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}
	if model.DeletedAt.Valid {
		entity.DeletedAt = &model.DeletedAt.Time
	}
	return entity
}
