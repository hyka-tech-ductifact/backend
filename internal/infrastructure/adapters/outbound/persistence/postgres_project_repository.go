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

// ProjectModel is the GORM-specific database representation.
// It is NOT a domain entity. It lives here because only this adapter cares about it.
type ProjectModel struct {
	ID          uuid.UUID `gorm:"primaryKey"`
	Name        string
	Address     string
	ManagerName string
	Phone       string
	Description string
	ClientID    uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt

	// GORM association to ClientModel for FK constraint and cascade delete.
	Client ClientModel `gorm:"foreignKey:ClientID"`
}

func (ProjectModel) TableName() string {
	return "projects"
}

// --- Repository implementation ---

// PostgresProjectRepository implements domain's ProjectRepository interface.
type PostgresProjectRepository struct {
	db *gorm.DB
}

func NewPostgresProjectRepository(db *gorm.DB) *PostgresProjectRepository {
	return &PostgresProjectRepository{db: db}
}

func (r *PostgresProjectRepository) Create(ctx context.Context, project *entities.Project) error {
	model := toProjectModel(project)
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *PostgresProjectRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Project, error) {
	var model ProjectModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}
	return toProjectEntity(&model), nil
}

func (r *PostgresProjectRepository) GetByIDForOwner(ctx context.Context, id uuid.UUID, ownerID uuid.UUID) (*entities.Project, error) {
	var model ProjectModel
	err := r.db.WithContext(ctx).
		Joins("JOIN clients ON clients.id = projects.client_id AND clients.deleted_at IS NULL").
		Where("projects.id = ? AND clients.user_id = ?", id, ownerID).
		First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, r.diagnoseProjectFailure(ctx, id, ownerID)
	}
	if err != nil {
		return nil, err
	}
	return toProjectEntity(&model), nil
}

func (r *PostgresProjectRepository) diagnoseProjectFailure(ctx context.Context, id uuid.UUID, ownerID uuid.UUID) error {
	var project ProjectModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&project).Error; err != nil {
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
		return repositories.ErrProjectNotOwned
	}
	return repositories.ErrProjectNotFound
}

func (r *PostgresProjectRepository) ListByClientID(ctx context.Context, clientID uuid.UUID, pg pagination.Pagination) ([]*entities.Project, int64, error) {
	var totalItems int64

	// Count total matching rows (before pagination)
	if err := r.db.WithContext(ctx).Model(&ProjectModel{}).Where("client_id = ?", clientID).Count(&totalItems).Error; err != nil {
		return nil, 0, err
	}

	// Fetch the requested page
	var models []ProjectModel
	err := r.db.WithContext(ctx).
		Where("client_id = ?", clientID).
		Order("created_at DESC").
		Offset((pg.Page - 1) * pg.PageSize).
		Limit(pg.PageSize).
		Find(&models).Error
	if err != nil {
		return nil, 0, err
	}

	projects := make([]*entities.Project, len(models))
	for i := range models {
		projects[i] = toProjectEntity(&models[i])
	}
	return projects, totalItems, nil
}

func (r *PostgresProjectRepository) Update(ctx context.Context, project *entities.Project) error {
	model := toProjectModel(project)
	return r.db.WithContext(ctx).Save(model).Error
}

func (r *PostgresProjectRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&ProjectModel{}, "id = ?", id).Error
}

// --- Mappers (package-level functions, not methods) ---

func toProjectModel(project *entities.Project) *ProjectModel {
	model := &ProjectModel{
		ID:          project.ID,
		Name:        project.Name,
		Address:     project.Address,
		ManagerName: project.ManagerName,
		Phone:       project.Phone,
		Description: project.Description,
		ClientID:    project.ClientID,
		CreatedAt:   project.CreatedAt,
		UpdatedAt:   project.UpdatedAt,
	}
	if project.DeletedAt != nil {
		model.DeletedAt = gorm.DeletedAt{Time: *project.DeletedAt, Valid: true}
	}
	return model
}

func toProjectEntity(model *ProjectModel) *entities.Project {
	entity := &entities.Project{
		ID:          model.ID,
		Name:        model.Name,
		Address:     model.Address,
		ManagerName: model.ManagerName,
		Phone:       model.Phone,
		Description: model.Description,
		ClientID:    model.ClientID,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}
	if model.DeletedAt.Valid {
		entity.DeletedAt = &model.DeletedAt.Time
	}
	return entity
}
