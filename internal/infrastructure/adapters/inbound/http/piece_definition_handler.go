package http

import (
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"ductifact/internal/application/usecases"
	"ductifact/internal/domain/entities"
	"ductifact/internal/infrastructure/adapters/inbound/http/helpers"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// --- DTOs (HTTP-specific, not domain objects) ---

// createPieceDefinitionData is the JSON portion of the multipart request.
type createPieceDefinitionData struct {
	Name            string   `json:"name"`
	DimensionSchema []string `json:"dimension_schema"`
}

// updatePieceDefinitionData is the JSON portion of the multipart update request.
type updatePieceDefinitionData struct {
	Name            *string   `json:"name,omitempty"`
	DimensionSchema *[]string `json:"dimension_schema,omitempty"`
}

type PieceDefinitionResponse struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	ImageURL        string   `json:"image_url"`
	ThumbnailURL    string   `json:"thumbnail_url"`
	DimensionSchema []string `json:"dimension_schema"`
	Predefined      bool     `json:"predefined"`
	ArchivedAt      *string  `json:"archived_at"`
}

type ListPieceDefinitionResponse struct {
	Data       []*PieceDefinitionResponse `json:"data"`
	Page       int                        `json:"page"`
	PageSize   int                        `json:"page_size"`
	TotalItems int64                      `json:"total_items"`
	TotalPages int                        `json:"total_pages"`
}

// --- Handler ---

type PieceDefinitionHandler struct {
	pieceDefService usecases.PieceDefinitionService
}

func NewPieceDefinitionHandler(pieceDefService usecases.PieceDefinitionService) *PieceDefinitionHandler {
	return &PieceDefinitionHandler{pieceDefService: pieceDefService}
}

// CreatePieceDefinition handles POST /piece-definitions (multipart/form-data)
// Parts: "data" (JSON string with name + dimension_schema), "image" (optional binary)
func (h *PieceDefinitionHandler) CreatePieceDefinition(c *gin.Context) {
	userID := helpers.MustGetUserID(c)
	if c.IsAborted() {
		return
	}

	// --- Parse "data" part (JSON string) ---
	dataStr := c.PostForm("data")
	if dataStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing 'data' part"})
		return
	}

	var data createPieceDefinitionData
	if err := json.Unmarshal([]byte(dataStr), &data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON in 'data' part: " + err.Error()})
		return
	}

	if data.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	if len(data.DimensionSchema) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dimension_schema is required"})
		return
	}

	// --- Parse optional "image" part ---
	var file *usecases.FileInput
	fh, err := c.FormFile("image")
	if err == nil {
		fi, err := parseAndValidateImage(fh)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		file = fi
	}

	def, err := h.pieceDefService.CreatePieceDefinition(c.Request.Context(), userID, entities.CreatePieceDefParams{
		Name:            data.Name,
		DimensionSchema: data.DimensionSchema,
	}, file)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toPieceDefResponse(def))
}

// ListPieceDefinitions handles GET /piece-definitions
func (h *PieceDefinitionHandler) ListPieceDefinitions(c *gin.Context) {
	userID := helpers.MustGetUserID(c)
	if c.IsAborted() {
		return
	}

	pg, err := parsePagination(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var includeArchived bool
	if raw, exists := c.GetQuery("include_archived"); exists {
		switch strings.ToLower(raw) {
		case "true":
			includeArchived = true
		case "false":
			includeArchived = false
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "include_archived must be a boolean (true/false)"})
			return
		}
	}

	result, err := h.pieceDefService.ListPieceDefinitions(c.Request.Context(), userID, includeArchived, pg)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response := make([]*PieceDefinitionResponse, len(result.Data))
	for i, def := range result.Data {
		response[i] = toPieceDefResponse(def)
	}

	c.JSON(http.StatusOK, &ListPieceDefinitionResponse{
		Data:       response,
		Page:       result.Page,
		PageSize:   result.PageSize,
		TotalItems: result.TotalItems,
		TotalPages: result.TotalPages,
	})
}

// GetPieceDefinition handles GET /piece-definitions/:piece_definition_id
func (h *PieceDefinitionHandler) GetPieceDefinition(c *gin.Context) {
	userID := helpers.MustGetUserID(c)
	if c.IsAborted() {
		return
	}

	defID, err := uuid.Parse(c.Param("piece_definition_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid piece definition ID format"})
		return
	}

	def, err := h.pieceDefService.GetPieceDefinitionByID(c.Request.Context(), defID, userID)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, toPieceDefResponse(def))
}

// UpdatePieceDefinition handles PUT /piece-definitions/:piece_definition_id (multipart/form-data)
// Parts: "data" (JSON string with optional name + dimension_schema), "image" (optional binary)
func (h *PieceDefinitionHandler) UpdatePieceDefinition(c *gin.Context) {
	userID := helpers.MustGetUserID(c)
	if c.IsAborted() {
		return
	}

	defID, err := uuid.Parse(c.Param("piece_definition_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid piece definition ID format"})
		return
	}

	params := entities.UpdatePieceDefParams{}

	// "data" part is optional for update (may only send image)
	dataStr := c.PostForm("data")
	if dataStr != "" {
		var data updatePieceDefinitionData
		if err := json.Unmarshal([]byte(dataStr), &data); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON in 'data' part: " + err.Error()})
			return
		}
		params.Name = data.Name
		params.DimensionSchema = data.DimensionSchema
	}

	// --- Parse optional "image" part ---
	var file *usecases.FileInput
	fh, err := c.FormFile("image")
	if err == nil {
		fi, err := parseAndValidateImage(fh)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		file = fi
	}

	def, err := h.pieceDefService.UpdatePieceDefinition(c.Request.Context(), defID, userID, params, file)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, toPieceDefResponse(def))
}

// DeletePieceDefinition handles DELETE /piece-definitions/:piece_definition_id
func (h *PieceDefinitionHandler) DeletePieceDefinition(c *gin.Context) {
	userID := helpers.MustGetUserID(c)
	if c.IsAborted() {
		return
	}

	defID, err := uuid.Parse(c.Param("piece_definition_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid piece definition ID format"})
		return
	}

	if err := h.pieceDefService.DeletePieceDefinition(c.Request.Context(), defID, userID); err != nil {
		helpers.HandleError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// ArchivePieceDefinition handles POST /piece-definitions/:piece_definition_id/archive
func (h *PieceDefinitionHandler) ArchivePieceDefinition(c *gin.Context) {
	userID := helpers.MustGetUserID(c)
	if c.IsAborted() {
		return
	}

	defID, err := uuid.Parse(c.Param("piece_definition_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid piece definition ID format"})
		return
	}

	def, err := h.pieceDefService.ArchivePieceDefinition(c.Request.Context(), defID, userID)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, toPieceDefResponse(def))
}

// UnarchivePieceDefinition handles POST /piece-definitions/:piece_definition_id/unarchive
func (h *PieceDefinitionHandler) UnarchivePieceDefinition(c *gin.Context) {
	userID := helpers.MustGetUserID(c)
	if c.IsAborted() {
		return
	}

	defID, err := uuid.Parse(c.Param("piece_definition_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid piece definition ID format"})
		return
	}

	def, err := h.pieceDefService.UnarchivePieceDefinition(c.Request.Context(), defID, userID)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, toPieceDefResponse(def))
}

// --- Mappers ---

// toFileURL converts a storage key to a public URL served by the file proxy.
// Returns empty string if the key is empty.
func toFileURL(storageKey string) string {
	if storageKey == "" {
		return ""
	}
	return fileProxyPrefix + storageKey
}

// deriveThumbnailURL converts an original key to its thumbnail counterpart by convention.
// "piece-definitions/{id}/original.png" → "/v1/files/piece-definitions/{id}/thumb.png"
// Returns empty string if there's no image.
func deriveThumbnailURL(imageURL string) string {
	if imageURL == "" {
		return ""
	}
	thumbKey := strings.Replace(imageURL, "/original", "/thumb", 1)
	return fileProxyPrefix + thumbKey
}

func toPieceDefResponse(def *entities.PieceDefinition) *PieceDefinitionResponse {
	var archivedAt *string
	if def.ArchivedAt != nil {
		formatted := def.ArchivedAt.Format(time.RFC3339)
		archivedAt = &formatted
	}

	return &PieceDefinitionResponse{
		ID:              def.ID.String(),
		Name:            def.Name,
		ImageURL:        toFileURL(def.ImageURL),
		ThumbnailURL:    deriveThumbnailURL(def.ImageURL),
		DimensionSchema: def.DimensionSchema,
		Predefined:      def.Predefined,
		ArchivedAt:      archivedAt,
	}
}

// --- Helpers ---

// parseAndValidateImage opens the multipart file, detects the real MIME type
// via magic bytes, and returns a FileInput ready for the service layer.
func parseAndValidateImage(fh *multipart.FileHeader) (*usecases.FileInput, error) {
	f, err := fh.Open()
	if err != nil {
		return nil, err
	}

	// Read first 512 bytes for MIME detection (http.DetectContentType uses up to 512)
	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil {
		f.Close()
		return nil, err
	}

	detectedType := http.DetectContentType(buf[:n])

	// Validate against allowed types
	switch detectedType {
	case "image/jpeg", "image/png", "image/webp":
		// OK
	default:
		f.Close()
		return nil, fmt.Errorf("unsupported image type %q: only JPEG, PNG and WebP are allowed", detectedType)
	}

	// Seek back to start so the service reads the full file
	if _, err := f.Seek(0, 0); err != nil {
		f.Close()
		return nil, err
	}

	return &usecases.FileInput{
		Reader:      f,
		Filename:    fh.Filename,
		ContentType: detectedType,
		Size:        fh.Size,
	}, nil
}
