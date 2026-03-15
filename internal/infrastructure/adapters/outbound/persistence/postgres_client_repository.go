package persistence

import (
	"context"
	"errors"
	"time"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// --- Database Model (infrastructure concern) ---

// ClientModel is the GORM-specific database representation.
// It is NOT a domain entity. It lives here because only this adapter cares about it.
type ClientModel struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name      string    `gorm:"not null"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	CreatedAt time.Time
	UpdatedAt time.Time

	// GORM association to UserModel for FK constraint and cascade delete.
	User UserModel `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
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

func (r *PostgresClientRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.Client, error) {
	var models []ClientModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&models).Error; err != nil {
		return nil, err
	}

	clients := make([]*entities.Client, len(models))
	for i := range models {
		clients[i] = toClientEntity(&models[i])
	}
	return clients, nil
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
	return &ClientModel{
		ID:        client.ID,
		Name:      client.Name,
		UserID:    client.UserID,
		CreatedAt: client.CreatedAt,
		UpdatedAt: client.UpdatedAt,
	}
}

func toClientEntity(model *ClientModel) *entities.Client {
	return &entities.Client{
		ID:        model.ID,
		Name:      model.Name,
		UserID:    model.UserID,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
}
