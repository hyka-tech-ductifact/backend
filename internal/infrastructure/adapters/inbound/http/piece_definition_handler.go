package http

import (
	"net/http"

	"ductifact/internal/application/usecases"
	"ductifact/internal/domain/entities"
	"ductifact/internal/infrastructure/adapters/inbound/http/helpers"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// --- DTOs (HTTP-specific, not domain objects) ---

type CreatePieceDefinitionRequest struct {
	Name            string   `json:"name" binding:"required"`
	ImageURL        string   `json:"image_url"`
	DimensionSchema []string `json:"dimension_schema" binding:"required,min=1"`
}

type UpdatePieceDefinitionRequest struct {
	Name            *string   `json:"name" binding:"omitempty"`
	ImageURL        *string   `json:"image_url"`
	DimensionSchema *[]string `json:"dimension_schema" binding:"omitempty"`
}

type PieceDefinitionResponse struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	ImageURL        string   `json:"image_url"`
	DimensionSchema []string `json:"dimension_schema"`
	Predefined      bool     `json:"predefined"`
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

// CreatePieceDefinition handles POST /users/me/piece-definitions
func (h *PieceDefinitionHandler) CreatePieceDefinition(c *gin.Context) {
	userID := helpers.MustGetUserID(c)
	if c.IsAborted() {
		return
	}

	var req CreatePieceDefinitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	def, err := h.pieceDefService.CreatePieceDefinition(c.Request.Context(), userID, entities.CreatePieceDefParams{
		Name:            req.Name,
		ImageURL:        req.ImageURL,
		DimensionSchema: req.DimensionSchema,
	})
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toPieceDefResponse(def))
}

// ListPieceDefinitions handles GET /users/me/piece-definitions
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

	result, err := h.pieceDefService.ListPieceDefinitions(c.Request.Context(), userID, pg)
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

// GetPieceDefinition handles GET /users/me/piece-definitions/:piece_definition_id
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

// UpdatePieceDefinition handles PUT /users/me/piece-definitions/:piece_definition_id
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

	var req UpdatePieceDefinitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	params := entities.UpdatePieceDefParams{
		Name:     req.Name,
		ImageURL: req.ImageURL,
	}
	if req.DimensionSchema != nil {
		params.DimensionSchema = req.DimensionSchema
	}

	def, err := h.pieceDefService.UpdatePieceDefinition(c.Request.Context(), defID, userID, params)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, toPieceDefResponse(def))
}

// DeletePieceDefinition handles DELETE /users/me/piece-definitions/:piece_definition_id
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

// --- Mappers ---

func toPieceDefResponse(def *entities.PieceDefinition) *PieceDefinitionResponse {
	return &PieceDefinitionResponse{
		ID:              def.ID.String(),
		Name:            def.Name,
		ImageURL:        def.ImageURL,
		DimensionSchema: def.DimensionSchema,
		Predefined:      def.Predefined,
	}
}
