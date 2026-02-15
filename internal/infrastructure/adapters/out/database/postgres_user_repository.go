package database

import (
	"context"
	"time"

	"backend-go-blueprint/internal/domain/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PostgresUserRepository struct {
	db *gorm.DB
}

type UserModel struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Email     string    `gorm:"not null"`
	Name      string    `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (UserModel) TableName() string {
	return "users"
}

func NewPostgresUserRepository(db *gorm.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) Create(ctx context.Context, user *entities.User) error {
	userModel := &UserModel{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
	return r.db.WithContext(ctx).Create(userModel).Error
}

func (r *PostgresUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	var userModel UserModel
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&userModel).Error
	if err != nil {
		return nil, err
	}
	return r.toDomainEntity(&userModel), nil
}

func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*entities.User, error) {
	var userModel UserModel
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&userModel).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return r.toDomainEntity(&userModel), nil
}

func (r *PostgresUserRepository) Update(ctx context.Context, user *entities.User) error {
	userModel := &UserModel{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		UpdatedAt: time.Now(),
	}
	return r.db.WithContext(ctx).Save(userModel).Error
}

func (r *PostgresUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&UserModel{}).Error
}

func (r *PostgresUserRepository) List(ctx context.Context, limit, offset int) ([]*entities.User, error) {
	var userModels []UserModel
	err := r.db.WithContext(ctx).Limit(limit).Offset(offset).Find(&userModels).Error
	if err != nil {
		return nil, err
	}

	users := make([]*entities.User, len(userModels))
	for i, model := range userModels {
		users[i] = r.toDomainEntity(&model)
	}
	return users, nil
}

func (r *PostgresUserRepository) toDomainEntity(model *UserModel) *entities.User {
	return &entities.User{
		ID:        model.ID,
		Email:     model.Email,
		Name:      model.Name,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
}