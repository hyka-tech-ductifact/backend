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
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name         string    `gorm:"not null"`
	Email        string    `gorm:"uniqueIndex;not null"`
	PasswordHash string    `gorm:"column:password_hash;not null;default:''"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
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
	return &UserModel{
		ID:           user.ID,
		Name:         user.Name,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}
}

func toUserEntity(model *UserModel) *entities.User {
	return &entities.User{
		ID:           model.ID,
		Name:         model.Name,
		Email:        model.Email,
		PasswordHash: model.PasswordHash,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}
}
