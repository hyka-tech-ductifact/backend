package services_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"ductifact/internal/application/ports"
	"ductifact/internal/application/services"
	"ductifact/internal/application/usecases"
	"ductifact/internal/domain/entities"
	"ductifact/test/unit/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newPieceDefServiceCustom creates a service with explicit mock dependencies.
func newPieceDefServiceCustom(
	repo *mocks.MockPieceDefinitionRepository,
	storage *mocks.MockFileStorage,
	imgProc *mocks.MockImageProcessor,
) usecases.PieceDefinitionService {
	return services.NewPieceDefinitionService(repo, storage, imgProc)
}

// fakeFileInput creates a FileInput suitable for testing.
func fakeFileInput(content string) *usecases.FileInput {
	data := []byte(content)
	return &usecases.FileInput{
		Reader:      bytes.NewReader(data),
		Filename:    "photo.png",
		ContentType: "image/png",
		Size:        int64(len(data)),
	}
}

// =============================================================================
// CreatePieceDefinition — Image Upload
// =============================================================================

func TestCreatePieceDefinition_WithImage_UploadsOriginalAndThumbnail(t *testing.T) {
	var uploadedKeys []string
	storage := &mocks.MockFileStorage{
		UploadFn: func(ctx context.Context, key string, reader io.Reader, contentType string, size int64) error {
			uploadedKeys = append(uploadedKeys, key)
			return nil
		},
	}
	svc := newPieceDefServiceCustom(&mocks.MockPieceDefinitionRepository{}, storage, &mocks.MockImageProcessor{})

	def, err := svc.CreatePieceDefinition(context.Background(), uuid.New(), entities.CreatePieceDefParams{
		Name:            "With Image",
		DimensionSchema: []string{"Length"},
	}, fakeFileInput("image-data"))

	require.NoError(t, err)
	assert.Len(t, uploadedKeys, 2)
	assert.True(t, strings.Contains(uploadedKeys[0], "/original"))
	assert.True(t, strings.Contains(uploadedKeys[1], "/thumb"))
	assert.NotEmpty(t, def.ImageURL)
	assert.True(t, strings.Contains(def.ImageURL, "/original"))
}

func TestCreatePieceDefinition_WithImage_SetsImageURLFromKey(t *testing.T) {
	svc := newPieceDefServiceCustom(&mocks.MockPieceDefinitionRepository{}, &mocks.MockFileStorage{}, &mocks.MockImageProcessor{})

	def, err := svc.CreatePieceDefinition(context.Background(), uuid.New(), entities.CreatePieceDefParams{
		Name:            "Keyed",
		DimensionSchema: []string{"W"},
	}, fakeFileInput("data"))

	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(def.ImageURL, "piece-definitions/"))
	assert.True(t, strings.HasSuffix(def.ImageURL, "/original.png"))
}

// =============================================================================
// Image Validation
// =============================================================================

func TestCreatePieceDefinition_WithUnsupportedImageType_ReturnsError(t *testing.T) {
	svc := newPieceDefService(&mocks.MockPieceDefinitionRepository{})

	file := fakeFileInput("data")
	file.ContentType = "application/pdf"

	def, err := svc.CreatePieceDefinition(context.Background(), uuid.New(), entities.CreatePieceDefParams{
		Name:            "Bad Type",
		DimensionSchema: []string{"L"},
	}, file)

	assert.Nil(t, def)
	assert.ErrorIs(t, err, services.ErrUnsupportedImageType)
}

func TestCreatePieceDefinition_WithOversizedImage_ReturnsError(t *testing.T) {
	svc := newPieceDefService(&mocks.MockPieceDefinitionRepository{})

	file := fakeFileInput("data")
	file.Size = 6 << 20 // 6 MB

	def, err := svc.CreatePieceDefinition(context.Background(), uuid.New(), entities.CreatePieceDefParams{
		Name:            "Too Big",
		DimensionSchema: []string{"L"},
	}, file)

	assert.Nil(t, def)
	assert.ErrorIs(t, err, services.ErrImageTooLarge)
}

// =============================================================================
// Compensation — rollback on failure
// =============================================================================

func TestCreatePieceDefinition_WhenThumbnailFailsWithDecodeError_ReturnsImageCorrupt(t *testing.T) {
	var deletedKeys []string
	storage := &mocks.MockFileStorage{
		UploadFn: func(ctx context.Context, key string, reader io.Reader, contentType string, size int64) error {
			return nil // original upload succeeds
		},
		DeleteFn: func(ctx context.Context, key string) error {
			deletedKeys = append(deletedKeys, key)
			return nil
		},
	}
	imgProc := &mocks.MockImageProcessor{
		GenerateThumbnailFn: func(src io.Reader) (io.Reader, string, int64, error) {
			return nil, "", 0, ports.ErrImageDecode
		},
	}
	svc := newPieceDefServiceCustom(&mocks.MockPieceDefinitionRepository{}, storage, imgProc)

	def, err := svc.CreatePieceDefinition(context.Background(), uuid.New(), entities.CreatePieceDefParams{
		Name:            "Thumb Fail",
		DimensionSchema: []string{"L"},
	}, fakeFileInput("data"))

	assert.Nil(t, def)
	assert.ErrorIs(t, err, services.ErrImageCorrupt)
	assert.Len(t, deletedKeys, 1, "original should be cleaned up")
	assert.True(t, strings.Contains(deletedKeys[0], "/original"))
}

func TestCreatePieceDefinition_WhenThumbnailFailsWithInternalError_ReturnsGenericError(t *testing.T) {
	var deletedKeys []string
	storage := &mocks.MockFileStorage{
		UploadFn: func(ctx context.Context, key string, reader io.Reader, contentType string, size int64) error {
			return nil
		},
		DeleteFn: func(ctx context.Context, key string) error {
			deletedKeys = append(deletedKeys, key)
			return nil
		},
	}
	imgProc := &mocks.MockImageProcessor{
		GenerateThumbnailFn: func(src io.Reader) (io.Reader, string, int64, error) {
			return nil, "", 0, errors.New("disk I/O error")
		},
	}
	svc := newPieceDefServiceCustom(&mocks.MockPieceDefinitionRepository{}, storage, imgProc)

	def, err := svc.CreatePieceDefinition(context.Background(), uuid.New(), entities.CreatePieceDefParams{
		Name:            "Internal Fail",
		DimensionSchema: []string{"L"},
	}, fakeFileInput("data"))

	assert.Nil(t, def)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, services.ErrImageCorrupt, "generic errors should not be mapped to ErrImageCorrupt")
	assert.Len(t, deletedKeys, 1, "original should be cleaned up")
}

func TestCreatePieceDefinition_WhenRepoFails_DeletesBothFiles(t *testing.T) {
	var deletedKeys []string
	storage := &mocks.MockFileStorage{
		UploadFn: func(ctx context.Context, key string, reader io.Reader, contentType string, size int64) error {
			return nil
		},
		DeleteFn: func(ctx context.Context, key string) error {
			deletedKeys = append(deletedKeys, key)
			return nil
		},
	}
	repo := &mocks.MockPieceDefinitionRepository{
		CreateFn: func(ctx context.Context, def *entities.PieceDefinition) error {
			return errors.New("db down")
		},
	}
	svc := newPieceDefServiceCustom(repo, storage, &mocks.MockImageProcessor{})

	def, err := svc.CreatePieceDefinition(context.Background(), uuid.New(), entities.CreatePieceDefParams{
		Name:            "DB Fail",
		DimensionSchema: []string{"L"},
	}, fakeFileInput("data"))

	assert.Nil(t, def)
	assert.EqualError(t, err, "db down")
	assert.Len(t, deletedKeys, 2, "both original and thumb should be cleaned up")
}

// =============================================================================
// UpdatePieceDefinition — Image Upload
// =============================================================================

func TestUpdatePieceDefinition_WithNewImage_UploadsAndSetsURL(t *testing.T) {
	userID := uuid.New()
	def := newTestPieceDef(userID)
	def.ImageURL = "piece-definitions/old-id/original.png"

	var uploadedKeys []string
	var deletedKeys []string
	storage := &mocks.MockFileStorage{
		UploadFn: func(ctx context.Context, key string, reader io.Reader, contentType string, size int64) error {
			uploadedKeys = append(uploadedKeys, key)
			return nil
		},
		DeleteFn: func(ctx context.Context, key string) error {
			deletedKeys = append(deletedKeys, key)
			return nil
		},
	}

	repo := pieceDefRepoReturning(def)
	svc := newPieceDefServiceCustom(repo, storage, &mocks.MockImageProcessor{})

	result, err := svc.UpdatePieceDefinition(context.Background(), def.ID, userID, entities.UpdatePieceDefParams{}, fakeFileInput("new-image"))

	require.NoError(t, err)
	assert.Len(t, uploadedKeys, 2, "new original + new thumb uploaded")
	assert.True(t, strings.Contains(result.ImageURL, "/original"))
	assert.NotEqual(t, "piece-definitions/old-id/original.png", result.ImageURL, "URL should be new")

	// Old files should be deleted after successful persist
	assert.Len(t, deletedKeys, 2, "old original + old thumb deleted")
}

// =============================================================================
// DeletePieceDefinition — Image Cleanup
// =============================================================================

func TestDeletePieceDefinition_WithImage_DeletesFiles(t *testing.T) {
	userID := uuid.New()
	def := newTestPieceDef(userID)
	def.ImageURL = "piece-definitions/some-id/original.png"

	var deletedKeys []string
	storage := &mocks.MockFileStorage{
		DeleteFn: func(ctx context.Context, key string) error {
			deletedKeys = append(deletedKeys, key)
			return nil
		},
	}
	repo := pieceDefRepoReturning(def)
	repo.DeleteFn = func(ctx context.Context, id uuid.UUID) error { return nil }

	svc := newPieceDefServiceCustom(repo, storage, &mocks.MockImageProcessor{})

	err := svc.DeletePieceDefinition(context.Background(), def.ID, userID)

	assert.NoError(t, err)
	assert.Len(t, deletedKeys, 2)
	assert.Contains(t, deletedKeys[0], "/original")
	assert.Contains(t, deletedKeys[1], "/thumb")
}

func TestDeletePieceDefinition_WithoutImage_SkipsFileCleanup(t *testing.T) {
	userID := uuid.New()
	def := newTestPieceDef(userID)
	def.ImageURL = "" // no image

	deleteCalled := false
	storage := &mocks.MockFileStorage{
		DeleteFn: func(ctx context.Context, key string) error {
			deleteCalled = true
			return nil
		},
	}
	repo := pieceDefRepoReturning(def)
	repo.DeleteFn = func(ctx context.Context, id uuid.UUID) error { return nil }

	svc := newPieceDefServiceCustom(repo, storage, &mocks.MockImageProcessor{})

	err := svc.DeletePieceDefinition(context.Background(), def.ID, userID)

	assert.NoError(t, err)
	assert.False(t, deleteCalled, "should not attempt file deletion when no image")
}
