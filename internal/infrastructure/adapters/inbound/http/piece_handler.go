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

type CreatePieceRequest struct {
	Title        string             `json:"title" binding:"required"`
	DefinitionID string             `json:"definition_id" binding:"required"`
	Dimensions   map[string]float64 `json:"dimensions" binding:"required"`
	Quantity     int                `json:"quantity" binding:"required,min=1"`
}

type UpdatePieceRequest struct {
	Title      *string             `json:"title" binding:"omitempty"`
	Dimensions *map[string]float64 `json:"dimensions"`
	Quantity   *int                `json:"quantity" binding:"omitempty,min=1"`
}

type PieceResponse struct {
	ID           string             `json:"id"`
	Title        string             `json:"title"`
	DefinitionID string             `json:"definition_id"`
	Dimensions   map[string]float64 `json:"dimensions"`
	Quantity     int                `json:"quantity"`
	OrderID      string             `json:"order_id"`
}

type ListPieceResponse struct {
	Data       []*PieceResponse `json:"data"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalItems int64            `json:"total_items"`
	TotalPages int              `json:"total_pages"`
}

// --- Handler ---

type PieceHandler struct {
	pieceService usecases.PieceService
}

func NewPieceHandler(pieceService usecases.PieceService) *PieceHandler {
	return &PieceHandler{pieceService: pieceService}
}

// CreatePiece handles POST /orders/:order_id/pieces
func (h *PieceHandler) CreatePiece(c *gin.Context) {
	userID := helpers.MustGetUserID(c)
	if c.IsAborted() {
		return
	}

	orderID, err := uuid.Parse(c.Param("order_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order ID format"})
		return
	}

	var req CreatePieceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	definitionID, err := uuid.Parse(req.DefinitionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid definition ID format"})
		return
	}

	piece, err := h.pieceService.CreatePiece(c.Request.Context(), userID, entities.CreatePieceParams{
		Title:        req.Title,
		OrderID:      orderID,
		DefinitionID: definitionID,
		Dimensions:   req.Dimensions,
		Quantity:     req.Quantity,
	})
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toPieceResponse(piece))
}

// ListPieces handles GET /orders/:order_id/pieces
func (h *PieceHandler) ListPieces(c *gin.Context) {
	userID := helpers.MustGetUserID(c)
	if c.IsAborted() {
		return
	}

	orderID, err := uuid.Parse(c.Param("order_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order ID format"})
		return
	}

	pg, err := parsePagination(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.pieceService.ListPiecesByOrderID(c.Request.Context(), orderID, userID, pg)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response := make([]*PieceResponse, len(result.Data))
	for i, piece := range result.Data {
		response[i] = toPieceResponse(piece)
	}

	c.JSON(http.StatusOK, &ListPieceResponse{
		Data:       response,
		Page:       result.Page,
		PageSize:   result.PageSize,
		TotalItems: result.TotalItems,
		TotalPages: result.TotalPages,
	})
}

// GetPiece handles GET /orders/:order_id/pieces/:piece_id
func (h *PieceHandler) GetPiece(c *gin.Context) {
	userID := helpers.MustGetUserID(c)
	if c.IsAborted() {
		return
	}

	pieceID, err := uuid.Parse(c.Param("piece_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid piece ID format"})
		return
	}

	piece, err := h.pieceService.GetPieceByID(c.Request.Context(), pieceID, userID)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, toPieceResponse(piece))
}

// UpdatePiece handles PUT /orders/:order_id/pieces/:piece_id
func (h *PieceHandler) UpdatePiece(c *gin.Context) {
	userID := helpers.MustGetUserID(c)
	if c.IsAborted() {
		return
	}

	pieceID, err := uuid.Parse(c.Param("piece_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid piece ID format"})
		return
	}

	var req UpdatePieceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	piece, err := h.pieceService.UpdatePiece(c.Request.Context(), pieceID, userID, entities.UpdatePieceParams{
		Title:      req.Title,
		Dimensions: req.Dimensions,
		Quantity:   req.Quantity,
	})
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, toPieceResponse(piece))
}

// DeletePiece handles DELETE /orders/:order_id/pieces/:piece_id
func (h *PieceHandler) DeletePiece(c *gin.Context) {
	userID := helpers.MustGetUserID(c)
	if c.IsAborted() {
		return
	}

	pieceID, err := uuid.Parse(c.Param("piece_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid piece ID format"})
		return
	}

	if err := h.pieceService.DeletePiece(c.Request.Context(), pieceID, userID); err != nil {
		helpers.HandleError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// --- Mapper: Domain → HTTP Response ---

func toPieceResponse(piece *entities.Piece) *PieceResponse {
	return &PieceResponse{
		ID:           piece.ID.String(),
		Title:        piece.Title,
		DefinitionID: piece.DefinitionID.String(),
		Dimensions:   piece.Dimensions,
		Quantity:     piece.Quantity,
		OrderID:      piece.OrderID.String(),
	}
}
