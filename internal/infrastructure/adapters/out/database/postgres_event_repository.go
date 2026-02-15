package database

import (
	"context"
	"time"

	"event-service/internal/domain/entities"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PostgresEventRepository struct {
	db *gorm.DB
}

type EventModel struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Title       string    `gorm:"not null"`
	Description string    `gorm:"type:text"`
	Location    string    `gorm:"not null"`
	StartTime   time.Time `gorm:"not null"`
	EndTime     time.Time `gorm:"not null"`
	OrganizerID uuid.UUID `gorm:"type:uuid;not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (EventModel) TableName() string {
	return "events"
}

func NewPostgresEventRepository(db *gorm.DB) *PostgresEventRepository {
	return &PostgresEventRepository{db: db}
}

func (r *PostgresEventRepository) Create(ctx context.Context, event *entities.Event) error {
	eventModel := &EventModel{
		ID:          event.ID,
		Title:       event.Title,
		Description: event.Description,
		Location:    event.Location,
		StartTime:   event.StartTime,
		EndTime:     event.EndTime,
		OrganizerID: event.OrganizerID,
		CreatedAt:   event.CreatedAt,
		UpdatedAt:   event.UpdatedAt,
	}
	return r.db.WithContext(ctx).Create(eventModel).Error
}

func (r *PostgresEventRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Event, error) {
	var eventModel EventModel
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&eventModel).Error
	if err != nil {
		return nil, err
	}
	return r.toDomainEntity(&eventModel), nil
}

func (r *PostgresEventRepository) GetByOrganizerID(ctx context.Context, organizerID uuid.UUID) ([]*entities.Event, error) {
	var eventModels []EventModel
	err := r.db.WithContext(ctx).Where("organizer_id = ?", organizerID).Find(&eventModels).Error
	if err != nil {
		return nil, err
	}

	events := make([]*entities.Event, len(eventModels))
	for i, model := range eventModels {
		events[i] = r.toDomainEntity(&model)
	}
	return events, nil
}

func (r *PostgresEventRepository) Update(ctx context.Context, event *entities.Event) error {
	eventModel := &EventModel{
		ID:          event.ID,
		Title:       event.Title,
		Description: event.Description,
		Location:    event.Location,
		StartTime:   event.StartTime,
		EndTime:     event.EndTime,
		OrganizerID: event.OrganizerID,
		UpdatedAt:   time.Now(),
	}
	return r.db.WithContext(ctx).Save(eventModel).Error
}

func (r *PostgresEventRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&EventModel{}).Error
}

func (r *PostgresEventRepository) List(ctx context.Context, limit, offset int) ([]*entities.Event, error) {
	var eventModels []EventModel
	err := r.db.WithContext(ctx).Limit(limit).Offset(offset).Find(&eventModels).Error
	if err != nil {
		return nil, err
	}

	events := make([]*entities.Event, len(eventModels))
	for i, model := range eventModels {
		events[i] = r.toDomainEntity(&model)
	}
	return events, nil
}

func (r *PostgresEventRepository) toDomainEntity(model *EventModel) *entities.Event {
	return &entities.Event{
		ID:          model.ID,
		Title:       model.Title,
		Description: model.Description,
		Location:    model.Location,
		StartTime:   model.StartTime,
		EndTime:     model.EndTime,
		OrganizerID: model.OrganizerID,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}
}
