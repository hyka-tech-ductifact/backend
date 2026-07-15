package persistence

import (
	"context"
	"errors"
	"time"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// --- Database Model ---

// OneTimeOTPModel is the GORM-specific database representation.
type OneTimeOTPModel struct {
	ID        uuid.UUID `gorm:"primaryKey"`
	Email     string
	Purpose   string
	CodeHash  string
	ExpiresAt time.Time
	Attempts  int
	CreatedAt time.Time
}

func (OneTimeOTPModel) TableName() string {
	return "one_time_otps"
}

// --- Repository implementation ---

// PostgresOneTimeOTPRepository implements repositories.OneTimeOTPRepository.
type PostgresOneTimeOTPRepository struct {
	db *gorm.DB
}

func NewPostgresOneTimeOTPRepository(db *gorm.DB) *PostgresOneTimeOTPRepository {
	return &PostgresOneTimeOTPRepository{db: db}
}

// Create upserts the OTP keyed by (email, purpose), replacing any previous code.
func (r *PostgresOneTimeOTPRepository) Create(ctx context.Context, otp *entities.OneTimeOTP) error {
	model := toOneTimeOTPModel(otp)
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "email"}, {Name: "purpose"}},
			DoUpdates: clause.AssignmentColumns([]string{"id", "code_hash", "expires_at", "attempts", "created_at"}),
		}).
		Create(model).
		Error
}

func (r *PostgresOneTimeOTPRepository) GetByEmailAndPurpose(
	ctx context.Context,
	email string,
	purpose entities.OTPPurpose,
) (*entities.OneTimeOTP, error) {
	var model OneTimeOTPModel
	if err := r.db.WithContext(ctx).
		Where("email = ? AND purpose = ?", email, string(purpose)).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}
	return toOneTimeOTPEntity(&model), nil
}

func (r *PostgresOneTimeOTPRepository) IncrementAttempts(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&OneTimeOTPModel{}).
		Where("id = ?", id).
		UpdateColumn("attempts", gorm.Expr("attempts + 1")).
		Error
}

func (r *PostgresOneTimeOTPRepository) DeleteByEmailAndPurpose(
	ctx context.Context,
	email string,
	purpose entities.OTPPurpose,
) error {
	return r.db.WithContext(ctx).
		Where("email = ? AND purpose = ?", email, string(purpose)).
		Delete(&OneTimeOTPModel{}).
		Error
}

// --- Mappers ---

func toOneTimeOTPModel(otp *entities.OneTimeOTP) *OneTimeOTPModel {
	return &OneTimeOTPModel{
		ID:        otp.ID,
		Email:     otp.Email,
		Purpose:   string(otp.Purpose),
		CodeHash:  otp.CodeHash,
		ExpiresAt: otp.ExpiresAt,
		Attempts:  otp.Attempts,
		CreatedAt: otp.CreatedAt,
	}
}

func toOneTimeOTPEntity(model *OneTimeOTPModel) *entities.OneTimeOTP {
	return &entities.OneTimeOTP{
		ID:        model.ID,
		Email:     model.Email,
		Purpose:   entities.OTPPurpose(model.Purpose),
		CodeHash:  model.CodeHash,
		ExpiresAt: model.ExpiresAt,
		Attempts:  model.Attempts,
		CreatedAt: model.CreatedAt,
	}
}
