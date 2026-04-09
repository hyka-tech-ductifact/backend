package http

import (
	"net/http"
	"strconv"

	"ductifact/internal/application/usecases"
	"ductifact/internal/domain/entities"
	"ductifact/internal/domain/pagination"
	"ductifact/internal/infrastructure/adapters/inbound/http/helpers"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// --- DTOs (HTTP-specific, not domain objects) ---

type CreateClientRequest struct {
	Name string `json:"name" binding:"required"`
}

type UpdateClientRequest struct {
	Name *string `json:"name" binding:"omitempty"`
}

type ClientResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	UserID string `json:"user_id"`
}

type ListClientResponse struct {
	Data       []*ClientResponse `json:"data"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
	TotalItems int64             `json:"total_items"`
	TotalPages int               `json:"total_pages"`
}

// --- Handler ---

type ClientHandler struct {
	clientService usecases.ClientService
}

func NewClientHandler(clientService usecases.ClientService) *ClientHandler {
	return &ClientHandler{clientService: clientService}
}

// CreateClient handles POST /users/me/clients
func (h *ClientHandler) CreateClient(c *gin.Context) {
	userID := helpers.MustGetUserID(c)
	if c.IsAborted() {
		return
	}

	var req CreateClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client, err := h.clientService.CreateClient(c.Request.Context(), req.Name, userID)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toClientResponse(client))
}

// ListClients handles GET /users/me/clients?page=1&page_size=20
func (h *ClientHandler) ListClients(c *gin.Context) {
	userID := helpers.MustGetUserID(c)
	if c.IsAborted() {
		return
	}

	pg, err := parsePagination(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.clientService.ListClientsByUserID(c.Request.Context(), userID, pg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	response := make([]*ClientResponse, len(result.Data))
	for i, client := range result.Data {
		response[i] = toClientResponse(client)
	}

	c.JSON(http.StatusOK, &ListClientResponse{
		Data:       response,
		Page:       result.Page,
		PageSize:   result.PageSize,
		TotalItems: result.TotalItems,
		TotalPages: result.TotalPages,
	})
}

// parsePagination extracts page and page_size from query params
// and returns a validation error if values are out of range.
func parsePagination(c *gin.Context) (pagination.Pagination, error) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	return pagination.NewPagination(page, pageSize)
}

// GetClient handles GET /users/me/clients/:client_id
func (h *ClientHandler) GetClient(c *gin.Context) {
	userID := helpers.MustGetUserID(c)
	if c.IsAborted() {
		return
	}

	id, err := uuid.Parse(c.Param("client_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client ID format"})
		return
	}

	client, err := h.clientService.GetClientByID(c.Request.Context(), id, userID)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, toClientResponse(client))
}

// UpdateClient handles PUT /users/me/clients/:client_id
func (h *ClientHandler) UpdateClient(c *gin.Context) {
	userID := helpers.MustGetUserID(c)
	if c.IsAborted() {
		return
	}

	id, err := uuid.Parse(c.Param("client_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client ID format"})
		return
	}

	var req UpdateClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client, err := h.clientService.UpdateClient(c.Request.Context(), id, userID, req.Name)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, toClientResponse(client))
}

// DeleteClient handles DELETE /users/me/clients/:client_id
func (h *ClientHandler) DeleteClient(c *gin.Context) {
	userID := helpers.MustGetUserID(c)
	if c.IsAborted() {
		return
	}

	id, err := uuid.Parse(c.Param("client_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client ID format"})
		return
	}

	if err := h.clientService.DeleteClient(c.Request.Context(), id, userID); err != nil {
		helpers.HandleError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// --- Mapper: Domain → HTTP Response ---

func toClientResponse(client *entities.Client) *ClientResponse {
	return &ClientResponse{
		ID:     client.ID.String(),
		Name:   client.Name,
		UserID: client.UserID.String(),
	}
}
