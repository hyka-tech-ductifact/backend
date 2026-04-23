package usecases

import (
	"context"
	"io"

	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/pagination"

	"github.com/google/uuid"
)

// FileInput carries an uploaded file through the application layer.
// The handler creates it; the service consumes it. nil = no file attached.
type FileInput struct {
	Reader      io.Reader // Stream of bytes (never nil when FileInput is present)
	Filename    string    // Original filename from the client
	ContentType string    // MIME type validated by magic bytes (e.g. "image/png")
	Size        int64     // File size in bytes
}

// PieceDefinitionService is the inbound port for piece definition operations.
// Inbound adapters (HTTP handlers, CLI, etc.) depend on this interface.
type PieceDefinitionService interface {
	CreatePieceDefinition(ctx context.Context, userID uuid.UUID, params entities.CreatePieceDefParams, file *FileInput) (*entities.PieceDefinition, error)
	GetPieceDefinitionByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*entities.PieceDefinition, error)
	ListPieceDefinitions(ctx context.Context, userID uuid.UUID, includeArchived bool, pg pagination.Pagination) (pagination.Result[*entities.PieceDefinition], error)
	UpdatePieceDefinition(ctx context.Context, id uuid.UUID, userID uuid.UUID, params entities.UpdatePieceDefParams, file *FileInput) (*entities.PieceDefinition, error)
	DeletePieceDefinition(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	ArchivePieceDefinition(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*entities.PieceDefinition, error)
	UnarchivePieceDefinition(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*entities.PieceDefinition, error)
}
