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

// PieceModel is the GORM-specific database representation.
// It is NOT a domain entity. It lives here because only this adapter cares about it.
type PieceModel struct {
	ID           uuid.UUID `gorm:"primaryKey"`
	Title        string
	OrderID      uuid.UUID
	DefinitionID uuid.UUID
	Dimensions   string // JSON string of map[string]float64
	Quantity     int
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt

	// GORM associations for FK constraints and cascade delete.
	Order      OrderModel           `gorm:"foreignKey:OrderID"`
	Definition PieceDefinitionModel `gorm:"foreignKey:DefinitionID"`
}

func (PieceModel) TableName() string {
	return "pieces"
}

// --- Repository implementation ---

// PostgresPieceRepository implements domain's PieceRepository interface.
type PostgresPieceRepository struct {
	db *gorm.DB
}

func NewPostgresPieceRepository(db *gorm.DB) *PostgresPieceRepository {
	return &PostgresPieceRepository{db: db}
}

func (r *PostgresPieceRepository) Create(ctx context.Context, piece *entities.Piece) error {
	model, err := toPieceModel(piece)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *PostgresPieceRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Piece, error) {
	var model PieceModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}
	return toPieceEntity(&model)
}

func (r *PostgresPieceRepository) GetByIDForOwner(ctx context.Context, id uuid.UUID, ownerID uuid.UUID) (*entities.Piece, error) {
	var model PieceModel
	err := r.db.WithContext(ctx).
		Joins("JOIN orders ON orders.id = pieces.order_id AND orders.deleted_at IS NULL").
		Joins("JOIN projects ON projects.id = orders.project_id AND projects.deleted_at IS NULL").
		Joins("JOIN clients ON clients.id = projects.client_id AND clients.deleted_at IS NULL").
		Where("pieces.id = ? AND clients.user_id = ?", id, ownerID).
		First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, r.diagnosePieceFailure(ctx, id, ownerID)
	}
	if err != nil {
		return nil, err
	}
	return toPieceEntity(&model)
}

func (r *PostgresPieceRepository) diagnosePieceFailure(ctx context.Context, id uuid.UUID, ownerID uuid.UUID) error {
	var piece PieceModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&piece).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return repositories.ErrPieceNotFound
		}
		return err
	}
	var order OrderModel
	if err := r.db.WithContext(ctx).Where("id = ?", piece.OrderID).First(&order).Error; err != nil {
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
		return repositories.ErrPieceNotOwned
	}
	return repositories.ErrPieceNotFound
}

func (r *PostgresPieceRepository) ListByOrderID(ctx context.Context, orderID uuid.UUID, pg pagination.Pagination) ([]*entities.Piece, int64, error) {
	var totalItems int64

	// Count total matching rows (before pagination)
	if err := r.db.WithContext(ctx).Model(&PieceModel{}).Where("order_id = ?", orderID).Count(&totalItems).Error; err != nil {
		return nil, 0, err
	}

	// Fetch the requested page
	var models []PieceModel
	err := r.db.WithContext(ctx).
		Where("order_id = ?", orderID).
		Order("created_at DESC").
		Offset((pg.Page - 1) * pg.PageSize).
		Limit(pg.PageSize).
		Find(&models).Error
	if err != nil {
		return nil, 0, err
	}

	pieces := make([]*entities.Piece, 0, len(models))
	for i := range models {
		piece, err := toPieceEntity(&models[i])
		if err != nil {
			return nil, 0, err
		}
		pieces = append(pieces, piece)
	}
	return pieces, totalItems, nil
}

func (r *PostgresPieceRepository) Update(ctx context.Context, piece *entities.Piece) error {
	model, err := toPieceModel(piece)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Save(model).Error
}

func (r *PostgresPieceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&PieceModel{}, "id = ?", id).Error
}

// --- Mappers (package-level functions, not methods) ---

func toPieceModel(piece *entities.Piece) (*PieceModel, error) {
	dimJSON, err := json.Marshal(piece.Dimensions)
	if err != nil {
		return nil, err
	}

	model := &PieceModel{
		ID:           piece.ID,
		Title:        piece.Title,
		OrderID:      piece.OrderID,
		DefinitionID: piece.DefinitionID,
		Dimensions:   string(dimJSON),
		Quantity:     piece.Quantity,
		CreatedAt:    piece.CreatedAt,
		UpdatedAt:    piece.UpdatedAt,
	}
	if piece.DeletedAt != nil {
		model.DeletedAt = gorm.DeletedAt{Time: *piece.DeletedAt, Valid: true}
	}
	return model, nil
}

func toPieceEntity(model *PieceModel) (*entities.Piece, error) {
	var dimensions map[string]float64
	if err := json.Unmarshal([]byte(model.Dimensions), &dimensions); err != nil {
		return nil, err
	}

	entity := &entities.Piece{
		ID:           model.ID,
		Title:        model.Title,
		OrderID:      model.OrderID,
		DefinitionID: model.DefinitionID,
		Dimensions:   dimensions,
		Quantity:     model.Quantity,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}
	if model.DeletedAt.Valid {
		entity.DeletedAt = &model.DeletedAt.Time
	}
	return entity, nil
}
