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

// --- Database Model ---

// OneTimeTokenModel is the GORM-specific database representation.
type OneTimeTokenModel struct {
	ID        uuid.UUID `gorm:"primaryKey"`
	UserID    uuid.UUID
	Token     string
	Type      string
	ExpiresAt time.Time
	CreatedAt time.Time
}

func (OneTimeTokenModel) TableName() string {
	return "one_time_tokens"
}

// --- Repository implementation ---

// PostgresOneTimeTokenRepository implements repositories.OneTimeTokenRepository.
type PostgresOneTimeTokenRepository struct {
	db *gorm.DB
}

func NewPostgresOneTimeTokenRepository(db *gorm.DB) *PostgresOneTimeTokenRepository {
	return &PostgresOneTimeTokenRepository{db: db}
}

func (r *PostgresOneTimeTokenRepository) Create(ctx context.Context, token *entities.OneTimeToken) error {
	model := toOneTimeTokenModel(token)
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *PostgresOneTimeTokenRepository) GetByToken(
	ctx context.Context,
	token string,
	tokenType entities.TokenType,
) (*entities.OneTimeToken, error) {
	var model OneTimeTokenModel
	if err := r.db.WithContext(ctx).Where("token = ? AND type = ?", token, string(tokenType)).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}
	return toOneTimeTokenEntity(&model), nil
}

func (r *PostgresOneTimeTokenRepository) DeleteByUserIDAndType(
	ctx context.Context,
	userID uuid.UUID,
	tokenType entities.TokenType,
) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND type = ?", userID, string(tokenType)).
		Delete(&OneTimeTokenModel{}).
		Error
}

// --- Mappers ---

func toOneTimeTokenModel(token *entities.OneTimeToken) *OneTimeTokenModel {
	return &OneTimeTokenModel{
		ID:        token.ID,
		UserID:    token.UserID,
		Token:     token.Token,
		Type:      string(token.Type),
		ExpiresAt: token.ExpiresAt,
		CreatedAt: token.CreatedAt,
	}
}

func toOneTimeTokenEntity(model *OneTimeTokenModel) *entities.OneTimeToken {
	return &entities.OneTimeToken{
		ID:        model.ID,
		UserID:    model.UserID,
		Token:     model.Token,
		Type:      entities.TokenType(model.Type),
		ExpiresAt: model.ExpiresAt,
		CreatedAt: model.CreatedAt,
	}
}
