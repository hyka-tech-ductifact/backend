package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/pagination"
	"ductifact/internal/domain/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// --- Database Model (infrastructure concern) ---

// PieceDefinitionModel is the GORM-specific database representation.
// It is NOT a domain entity. It lives here because only this adapter cares about it.
type PieceDefinitionModel struct {
	ID              uuid.UUID `gorm:"primaryKey"`
	Name            string
	ImageURL        string
	DimensionSchema string // JSON string of []string
	Predefined      bool
	UserID          *uuid.UUID
	ArchivedAt      *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       gorm.DeletedAt

	// GORM association to UserModel for FK constraint (nullable).
	User *UserModel `gorm:"foreignKey:UserID"`
}

func (PieceDefinitionModel) TableName() string {
	return "piece_definitions"
}

// --- Repository implementation ---

// PostgresPieceDefinitionRepository implements domain's PieceDefinitionRepository interface.
type PostgresPieceDefinitionRepository struct {
	db *gorm.DB
}

func NewPostgresPieceDefinitionRepository(db *gorm.DB) *PostgresPieceDefinitionRepository {
	return &PostgresPieceDefinitionRepository{db: db}
}

func (r *PostgresPieceDefinitionRepository) Create(ctx context.Context, def *entities.PieceDefinition) error {
	model, err := toPieceDefModel(def)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *PostgresPieceDefinitionRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.PieceDefinition, error) {
	var model PieceDefinitionModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}
	return toPieceDefEntity(&model)
}

// GetByIDForOwner returns a piece definition that is either predefined (visible
// to everyone) or owned by the given user. Returns a specific error for diagnostics.
func (r *PostgresPieceDefinitionRepository) GetByIDForOwner(ctx context.Context, id uuid.UUID, ownerID uuid.UUID) (*entities.PieceDefinition, error) {
	var model PieceDefinitionModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND (predefined = ? OR user_id = ?)", id, true, ownerID).
		First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, r.diagnosePieceDefFailure(ctx, id, ownerID)
	}
	if err != nil {
		return nil, err
	}
	return toPieceDefEntity(&model)
}

func (r *PostgresPieceDefinitionRepository) diagnosePieceDefFailure(ctx context.Context, id uuid.UUID, ownerID uuid.UUID) error {
	var count int64
	if err := r.db.WithContext(ctx).Model(&PieceDefinitionModel{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return repositories.ErrPieceDefNotFound
	}
	return repositories.ErrPieceDefNotOwned
}

// ListByUserID returns all predefined definitions + custom definitions created by the user.
// When includeArchived is false, archived definitions are excluded.
func (r *PostgresPieceDefinitionRepository) ListByUserID(ctx context.Context, userID uuid.UUID, includeArchived bool, pg pagination.Pagination) ([]*entities.PieceDefinition, int64, error) {
	var totalItems int64

	query := r.db.WithContext(ctx).Model(&PieceDefinitionModel{}).
		Where("predefined = ? OR user_id = ?", true, userID)

	if !includeArchived {
		query = query.Where("archived_at IS NULL")
	}

	if err := query.Count(&totalItems).Error; err != nil {
		return nil, 0, err
	}

	var models []PieceDefinitionModel
	err := query.
		Order("predefined DESC, created_at DESC").
		Offset((pg.Page - 1) * pg.PageSize).
		Limit(pg.PageSize).
		Find(&models).Error
	if err != nil {
		return nil, 0, err
	}

	defs := make([]*entities.PieceDefinition, 0, len(models))
	for i := range models {
		def, err := toPieceDefEntity(&models[i])
		if err != nil {
			return nil, 0, err
		}
		defs = append(defs, def)
	}
	return defs, totalItems, nil
}

func (r *PostgresPieceDefinitionRepository) Update(ctx context.Context, def *entities.PieceDefinition) error {
	model, err := toPieceDefModel(def)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Save(model).Error
}

func (r *PostgresPieceDefinitionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&PieceDefinitionModel{}, "id = ?", id).Error
}

// Archive sets archived_at to the current time, effectively disabling the definition.
func (r *PostgresPieceDefinitionRepository) Archive(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&PieceDefinitionModel{}).
		Where("id = ?", id).
		Update("archived_at", now).Error
}

// Unarchive clears archived_at, re-enabling the definition.
func (r *PostgresPieceDefinitionRepository) Unarchive(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&PieceDefinitionModel{}).
		Where("id = ?", id).
		Update("archived_at", nil).Error
}

// --- Mappers (package-level functions, not methods) ---

func toPieceDefModel(def *entities.PieceDefinition) (*PieceDefinitionModel, error) {
	schemaJSON, err := json.Marshal(def.DimensionSchema)
	if err != nil {
		return nil, err
	}

	model := &PieceDefinitionModel{
		ID:              def.ID,
		Name:            def.Name,
		ImageURL:        def.ImageURL,
		DimensionSchema: string(schemaJSON),
		Predefined:      def.Predefined,
		UserID:          def.UserID,
		ArchivedAt:      def.ArchivedAt,
		CreatedAt:       def.CreatedAt,
		UpdatedAt:       def.UpdatedAt,
	}
	if def.DeletedAt != nil {
		model.DeletedAt = gorm.DeletedAt{Time: *def.DeletedAt, Valid: true}
	}
	return model, nil
}

func toPieceDefEntity(model *PieceDefinitionModel) (*entities.PieceDefinition, error) {
	var schema []string
	if err := json.Unmarshal([]byte(model.DimensionSchema), &schema); err != nil {
		return nil, err
	}

	entity := &entities.PieceDefinition{
		ID:              model.ID,
		Name:            model.Name,
		ImageURL:        model.ImageURL,
		DimensionSchema: schema,
		Predefined:      model.Predefined,
		UserID:          model.UserID,
		ArchivedAt:      model.ArchivedAt,
		CreatedAt:       model.CreatedAt,
		UpdatedAt:       model.UpdatedAt,
	}
	if model.DeletedAt.Valid {
		entity.DeletedAt = &model.DeletedAt.Time
	}
	return entity, nil
}
