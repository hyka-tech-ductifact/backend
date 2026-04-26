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

// UserModel is the GORM-specific database representation.
// It is NOT a domain entity. It lives here because only this adapter cares about it.
type UserModel struct {
	ID           uuid.UUID `gorm:"primaryKey"`
	Name         string
	Email        string
	PasswordHash string
	Locale       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt
}

func (UserModel) TableName() string {
	return "users"
}

// --- Repository implementation ---

// PostgresUserRepository implements domain's UserRepository interface.
type PostgresUserRepository struct {
	db *gorm.DB
}

func NewPostgresUserRepository(db *gorm.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) Create(ctx context.Context, user *entities.User) error {
	model := toUserModel(user)
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *PostgresUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	var model UserModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}
	return toUserEntity(&model), nil
}

func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*entities.User, error) {
	var model UserModel
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}
	return toUserEntity(&model), nil
}

func (r *PostgresUserRepository) Update(ctx context.Context, user *entities.User) error {
	model := toUserModel(user)
	return r.db.WithContext(ctx).Save(model).Error
}

// --- Mappers (package-level functions, not methods) ---

func toUserModel(user *entities.User) *UserModel {
	model := &UserModel{
		ID:           user.ID,
		Name:         user.Name,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		Locale:       user.Locale,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}
	if user.DeletedAt != nil {
		model.DeletedAt = gorm.DeletedAt{Time: *user.DeletedAt, Valid: true}
	}
	return model
}

func toUserEntity(model *UserModel) *entities.User {
	entity := &entities.User{
		ID:           model.ID,
		Name:         model.Name,
		Email:        model.Email,
		PasswordHash: model.PasswordHash,
		Locale:       model.Locale,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}
	if model.DeletedAt.Valid {
		entity.DeletedAt = &model.DeletedAt.Time
	}
	return entity
}
