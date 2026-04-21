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

// ClientModel is the GORM-specific database representation.
// It is NOT a domain entity. It lives here because only this adapter cares about it.
type ClientModel struct {
	ID          uuid.UUID `gorm:"primaryKey"`
	Name        string
	Phone       string
	Email       string
	Description string
	UserID      uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt

	// GORM association to UserModel for FK constraint and cascade delete.
	User UserModel `gorm:"foreignKey:UserID"`
}

func (ClientModel) TableName() string {
	return "clients"
}

// --- Repository implementation ---

// PostgresClientRepository implements domain's ClientRepository interface.
type PostgresClientRepository struct {
	db *gorm.DB
}

func NewPostgresClientRepository(db *gorm.DB) *PostgresClientRepository {
	return &PostgresClientRepository{db: db}
}

func (r *PostgresClientRepository) Create(ctx context.Context, client *entities.Client) error {
	model := toClientModel(client)
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *PostgresClientRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Client, error) {
	var model ClientModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}
	return toClientEntity(&model), nil
}

func (r *PostgresClientRepository) GetByIDForOwner(ctx context.Context, id uuid.UUID, ownerID uuid.UUID) (*entities.Client, error) {
	var model ClientModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, ownerID).
		First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, r.diagnoseClientFailure(ctx, id, ownerID)
	}
	if err != nil {
		return nil, err
	}
	return toClientEntity(&model), nil
}

func (r *PostgresClientRepository) diagnoseClientFailure(ctx context.Context, id uuid.UUID, ownerID uuid.UUID) error {
	var count int64
	if err := r.db.WithContext(ctx).Model(&ClientModel{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return repositories.ErrClientNotFound
	}
	return repositories.ErrClientNotOwned
}

func (r *PostgresClientRepository) ListByUserID(ctx context.Context, userID uuid.UUID, pg pagination.Pagination) ([]*entities.Client, int64, error) {
	var totalItems int64

	// Count total matching rows (before pagination)
	if err := r.db.WithContext(ctx).Model(&ClientModel{}).Where("user_id = ?", userID).Count(&totalItems).Error; err != nil {
		return nil, 0, err
	}

	// Fetch the requested page
	var models []ClientModel
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset((pg.Page - 1) * pg.PageSize).
		Limit(pg.PageSize).
		Find(&models).Error
	if err != nil {
		return nil, 0, err
	}

	clients := make([]*entities.Client, len(models))
	for i := range models {
		clients[i] = toClientEntity(&models[i])
	}
	return clients, totalItems, nil
}

func (r *PostgresClientRepository) Update(ctx context.Context, client *entities.Client) error {
	model := toClientModel(client)
	return r.db.WithContext(ctx).Save(model).Error
}

func (r *PostgresClientRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&ClientModel{}, "id = ?", id).Error
}

// --- Mappers (package-level functions, not methods) ---

func toClientModel(client *entities.Client) *ClientModel {
	model := &ClientModel{
		ID:          client.ID,
		Name:        client.Name,
		Phone:       client.Phone,
		Email:       client.Email,
		Description: client.Description,
		UserID:      client.UserID,
		CreatedAt:   client.CreatedAt,
		UpdatedAt:   client.UpdatedAt,
	}
	if client.DeletedAt != nil {
		model.DeletedAt = gorm.DeletedAt{Time: *client.DeletedAt, Valid: true}
	}
	return model
}

func toClientEntity(model *ClientModel) *entities.Client {
	entity := &entities.Client{
		ID:          model.ID,
		Name:        model.Name,
		Phone:       model.Phone,
		Email:       model.Email,
		Description: model.Description,
		UserID:      model.UserID,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}
	if model.DeletedAt.Valid {
		entity.DeletedAt = &model.DeletedAt.Time
	}
	return entity
}
