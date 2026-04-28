package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"ductifact/internal/application/ports"
	"ductifact/internal/application/usecases"
	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/pagination"
	"ductifact/internal/domain/repositories"

	"github.com/google/uuid"
)

// --- Application-level errors ---

var (
	ErrPieceDefPredefined   = errors.New("predefined piece definitions cannot be modified")
	ErrUnsupportedImageType = errors.New("unsupported image type: only JPEG, PNG and WebP are allowed")
	ErrImageTooLarge        = errors.New("image exceeds the maximum allowed size of 5 MB")
	ErrImageCorrupt         = errors.New("image is corrupt or cannot be decoded")
	ErrPieceDefInUse        = errors.New("piece definition is in use by existing pieces")
	ErrPieceDefArchived     = errors.New("piece definition is archived")
)

const (
	maxImageSize = 5 << 20 // 5 MB
	bucketPrefix = "piece-definitions"
)

// pieceDefinitionService implements usecases.PieceDefinitionService.
// Unexported struct: can only be created via NewPieceDefinitionService.
type pieceDefinitionService struct {
	pieceDefRepo   repositories.PieceDefinitionRepository
	fileStorage    ports.FileStorage
	imageProcessor ports.ImageProcessor
	pieceRepo      repositories.PieceRepository
}

// NewPieceDefinitionService creates a new PieceDefinitionService.
func NewPieceDefinitionService(
	pieceDefRepo repositories.PieceDefinitionRepository,
	fileStorage ports.FileStorage,
	imageProcessor ports.ImageProcessor,
	pieceRepo repositories.PieceRepository,
) *pieceDefinitionService {
	return &pieceDefinitionService{
		pieceDefRepo:   pieceDefRepo,
		fileStorage:    fileStorage,
		imageProcessor: imageProcessor,
		pieceRepo:      pieceRepo,
	}
}

// CreatePieceDefinition orchestrates piece definition creation:
// 1. Upload image + thumbnail to storage (if file provided).
// 2. Build the domain entity (which validates all fields).
// 3. Persist via repository.
// On failure: compensate by deleting uploaded files.
func (s *pieceDefinitionService) CreatePieceDefinition(
	ctx context.Context,
	userID uuid.UUID,
	params entities.CreatePieceDefParams,
	file *usecases.FileInput,
) (*entities.PieceDefinition, error) {
	// Ensure the creator is set from the authenticated user
	params.UserID = userID

	var keys *uploadedKeys

	// --- Upload image if provided ---
	if file != nil {
		k, err := s.uploadWithThumbnail(ctx, file)
		if err != nil {
			return nil, err
		}
		keys = k
		params.ImageURL = keys.original
	}

	// Domain entity validates its own invariants
	def, err := entities.NewPieceDefinition(params)
	if err != nil {
		keys.rollback(ctx, s)
		return nil, err
	}

	if err := s.pieceDefRepo.Create(ctx, def); err != nil {
		keys.rollback(ctx, s)
		return nil, err
	}

	return def, nil
}

// GetPieceDefinitionByID retrieves a piece definition by ID.
// Returns the definition only if it is predefined or belongs to the requesting user.
func (s *pieceDefinitionService) GetPieceDefinitionByID(
	ctx context.Context,
	id uuid.UUID,
	userID uuid.UUID,
) (*entities.PieceDefinition, error) {
	return s.pieceDefRepo.GetByIDForOwner(ctx, id, userID)
}

// ListPieceDefinitions retrieves a paginated list of piece definitions visible to the user.
// This includes all predefined definitions + the user's custom definitions.
// When includeArchived is false, archived definitions are excluded.
func (s *pieceDefinitionService) ListPieceDefinitions(
	ctx context.Context,
	userID uuid.UUID,
	includeArchived bool,
	pg pagination.Pagination,
) (pagination.Result[*entities.PieceDefinition], error) {
	defs, totalItems, err := s.pieceDefRepo.ListByUserID(ctx, userID, includeArchived, pg)
	if err != nil {
		return pagination.Result[*entities.PieceDefinition]{}, err
	}

	return pagination.NewResult(defs, pg, totalItems), nil
}

// UpdatePieceDefinition applies a partial update to an existing piece definition.
// Only custom (non-predefined) definitions owned by the user can be updated.
// If a new image is provided, the old one is replaced (old files deleted).
func (s *pieceDefinitionService) UpdatePieceDefinition(
	ctx context.Context,
	id uuid.UUID,
	userID uuid.UUID,
	params entities.UpdatePieceDefParams,
	file *usecases.FileInput,
) (*entities.PieceDefinition, error) {
	def, err := s.pieceDefRepo.GetByIDForOwner(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	if def.Predefined {
		return nil, ErrPieceDefPredefined
	}

	// Prevent editing a definition that has pieces referencing it
	count, err := s.pieceRepo.CountByDefinitionID(ctx, def.ID)
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrPieceDefInUse
	}

	// --- Upload new image if provided ---
	var keys *uploadedKeys
	oldImageURL := def.ImageURL

	if file != nil {
		k, err := s.uploadWithThumbnail(ctx, file)
		if err != nil {
			return nil, err
		}
		keys = k
		imageURL := keys.original
		params.ImageURL = &imageURL
	}

	if !params.HasChanges() && file == nil {
		return def, nil
	}

	if params.Name != nil {
		if err := def.SetName(*params.Name); err != nil {
			return nil, err
		}
	}
	if params.ImageURL != nil {
		def.SetImageURL(*params.ImageURL)
	}
	if params.DimensionSchema != nil {
		if err := def.SetDimensionSchema(*params.DimensionSchema); err != nil {
			return nil, err
		}
	}

	def.UpdatedAt = time.Now()
	if err := s.pieceDefRepo.Update(ctx, def); err != nil {
		// Compensate: delete newly uploaded files if DB update fails
		keys.rollback(ctx, s)
		return nil, err
	}

	// --- Delete old files after successful persist ---
	if file != nil && oldImageURL != "" {
		s.compensateDelete(ctx, oldImageURL)
		oldThumbKey := strings.Replace(oldImageURL, "/original", "/thumb", 1)
		s.compensateDelete(ctx, oldThumbKey)
	}

	return def, nil
}

// DeletePieceDefinition removes a piece definition and its associated images.
// Only custom (non-predefined) definitions owned by the user can be deleted.
// Returns ErrPieceDefInUse if any pieces reference this definition.
func (s *pieceDefinitionService) DeletePieceDefinition(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	def, err := s.pieceDefRepo.GetByIDForOwner(ctx, id, userID)
	if err != nil {
		return err
	}

	if def.Predefined {
		return ErrPieceDefPredefined
	}

	count, err := s.pieceRepo.CountByDefinitionID(ctx, def.ID)
	if err != nil {
		return err
	}

	if count > 0 {
		return ErrPieceDefInUse
	}

	if err := s.pieceDefRepo.Delete(ctx, def.ID); err != nil {
		return err
	}

	// Best-effort cleanup of stored images after successful DB delete
	if def.ImageURL != "" {
		s.compensateDelete(ctx, def.ImageURL)
		thumbKey := strings.Replace(def.ImageURL, "/original", "/thumb", 1)
		s.compensateDelete(ctx, thumbKey)
	}

	return nil
}

// ArchivePieceDefinition sets the archived_at timestamp on a piece definition.
// Only custom (non-predefined) definitions owned by the user can be archived.
func (s *pieceDefinitionService) ArchivePieceDefinition(
	ctx context.Context,
	id uuid.UUID,
	userID uuid.UUID,
) (*entities.PieceDefinition, error) {
	def, err := s.pieceDefRepo.GetByIDForOwner(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	if def.Predefined {
		return nil, ErrPieceDefPredefined
	}

	if def.IsArchived() {
		return def, nil // already archived, idempotent
	}

	if err := s.pieceDefRepo.Archive(ctx, def.ID); err != nil {
		return nil, err
	}

	// Reflect the change in the returned entity
	now := time.Now()
	def.ArchivedAt = &now
	return def, nil
}

// UnarchivePieceDefinition clears the archived_at timestamp on a piece definition.
// Only custom (non-predefined) definitions owned by the user can be unarchived.
func (s *pieceDefinitionService) UnarchivePieceDefinition(
	ctx context.Context,
	id uuid.UUID,
	userID uuid.UUID,
) (*entities.PieceDefinition, error) {
	def, err := s.pieceDefRepo.GetByIDForOwner(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	if def.Predefined {
		return nil, ErrPieceDefPredefined
	}

	if !def.IsArchived() {
		return def, nil // already active, idempotent
	}

	if err := s.pieceDefRepo.Unarchive(ctx, def.ID); err != nil {
		return nil, err
	}

	def.ArchivedAt = nil
	return def, nil
}

// --- helpers ---

// uploadedKeys holds the storage keys generated by uploadWithThumbnail.
// nil-safe: calling rollback on a nil pointer is a no-op.
type uploadedKeys struct {
	original string
	thumb    string
}

// rollback deletes both uploaded files (best-effort). Safe to call on nil.
func (k *uploadedKeys) rollback(ctx context.Context, s *pieceDefinitionService) {
	if k == nil {
		return
	}
	s.compensateDelete(ctx, k.original)
	s.compensateDelete(ctx, k.thumb)
}

// uploadWithThumbnail validates the image, uploads the original and a thumbnail
// to storage, and returns the generated keys. On partial failure it compensates
// by deleting whatever was already uploaded.
func (s *pieceDefinitionService) uploadWithThumbnail(
	ctx context.Context,
	file *usecases.FileInput,
) (*uploadedKeys, error) {
	if err := validateImage(file); err != nil {
		return nil, err
	}

	id := uuid.New()
	ext := normalizeExtension(file.Filename)
	keys := &uploadedKeys{
		original: fmt.Sprintf("%s/%s/original%s", bucketPrefix, id, ext),
		thumb:    fmt.Sprintf("%s/%s/thumb%s", bucketPrefix, id, ext),
	}

	// Read the entire file into memory so we can feed it to both
	// the original upload and the thumbnail generator.
	data, err := io.ReadAll(file.Reader)
	if err != nil {
		return nil, fmt.Errorf("reading uploaded file: %w", err)
	}

	// Upload original
	if err := s.fileStorage.Upload(ctx, keys.original, bytes.NewReader(data), file.ContentType, int64(len(data))); err != nil {
		return nil, fmt.Errorf("uploading original image: %w", err)
	}

	// Generate and upload thumbnail
	thumbReader, thumbContentType, thumbSize, err := s.imageProcessor.GenerateThumbnail(bytes.NewReader(data))
	if err != nil {
		s.compensateDelete(ctx, keys.original)
		if errors.Is(err, ports.ErrImageDecode) {
			return nil, ErrImageCorrupt
		}
		return nil, fmt.Errorf("generating thumbnail: %w", err)
	}

	if err := s.fileStorage.Upload(ctx, keys.thumb, thumbReader, thumbContentType, thumbSize); err != nil {
		s.compensateDelete(ctx, keys.original)
		return nil, fmt.Errorf("uploading thumbnail: %w", err)
	}

	return keys, nil
}

// compensateDelete is a best-effort cleanup. Errors are logged but not propagated.
func (s *pieceDefinitionService) compensateDelete(ctx context.Context, key string) {
	if err := s.fileStorage.Delete(ctx, key); err != nil {
		slog.Warn("compensation: failed to delete file", "key", key, "error", err)
	}
}

// validateImage checks file size and content type.
func validateImage(file *usecases.FileInput) error {
	if file.Size > maxImageSize {
		return ErrImageTooLarge
	}
	switch file.ContentType {
	case "image/jpeg", "image/png", "image/webp":
		return nil
	default:
		return ErrUnsupportedImageType
	}
}

// normalizeExtension extracts and normalizes the file extension.
// Falls back to ".jpg" if no extension is present.
func normalizeExtension(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		ext = ".jpg"
	}
	return ext
}
