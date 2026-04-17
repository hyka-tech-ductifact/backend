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

type CreateOrderRequest struct {
	Title       string `json:"title" binding:"required"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

type UpdateOrderRequest struct {
	Title       *string `json:"title" binding:"omitempty"`
	Status      *string `json:"status"`
	Description *string `json:"description"`
}

type OrderResponse struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	Description string `json:"description"`
	ProjectID   string `json:"project_id"`
}

type ListOrderResponse struct {
	Data       []*OrderResponse `json:"data"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalItems int64            `json:"total_items"`
	TotalPages int              `json:"total_pages"`
}

// --- Handler ---

type OrderHandler struct {
	orderService usecases.OrderService
}

func NewOrderHandler(orderService usecases.OrderService) *OrderHandler {
	return &OrderHandler{orderService: orderService}
}

// CreateOrder handles POST /projects/:project_id/orders
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	userID := helpers.MustGetUserID(c)
	if c.IsAborted() {
		return
	}

	projectID, err := uuid.Parse(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project ID format"})
		return
	}

	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, err := h.orderService.CreateOrder(c.Request.Context(), userID, entities.CreateOrderParams{
		Title:       req.Title,
		Status:      req.Status,
		Description: req.Description,
		ProjectID:   projectID,
	})
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toOrderResponse(order))
}

// ListOrders handles GET /projects/:project_id/orders
func (h *OrderHandler) ListOrders(c *gin.Context) {
	userID := helpers.MustGetUserID(c)
	if c.IsAborted() {
		return
	}

	projectID, err := uuid.Parse(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project ID format"})
		return
	}

	pg, err := parsePagination(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.orderService.ListOrdersByProjectID(c.Request.Context(), projectID, userID, pg)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	response := make([]*OrderResponse, len(result.Data))
	for i, order := range result.Data {
		response[i] = toOrderResponse(order)
	}

	c.JSON(http.StatusOK, &ListOrderResponse{
		Data:       response,
		Page:       result.Page,
		PageSize:   result.PageSize,
		TotalItems: result.TotalItems,
		TotalPages: result.TotalPages,
	})
}

// GetOrder handles GET /orders/:order_id
func (h *OrderHandler) GetOrder(c *gin.Context) {
	userID := helpers.MustGetUserID(c)
	if c.IsAborted() {
		return
	}

	orderID, err := uuid.Parse(c.Param("order_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order ID format"})
		return
	}

	order, err := h.orderService.GetOrderByID(c.Request.Context(), orderID, userID)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, toOrderResponse(order))
}

// UpdateOrder handles PUT /orders/:order_id
func (h *OrderHandler) UpdateOrder(c *gin.Context) {
	userID := helpers.MustGetUserID(c)
	if c.IsAborted() {
		return
	}

	orderID, err := uuid.Parse(c.Param("order_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order ID format"})
		return
	}

	var req UpdateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, err := h.orderService.UpdateOrder(c.Request.Context(), orderID, userID, entities.UpdateOrderParams{
		Title:       req.Title,
		Status:      req.Status,
		Description: req.Description,
	})
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, toOrderResponse(order))
}

// DeleteOrder handles DELETE /orders/:order_id
func (h *OrderHandler) DeleteOrder(c *gin.Context) {
	userID := helpers.MustGetUserID(c)
	if c.IsAborted() {
		return
	}

	orderID, err := uuid.Parse(c.Param("order_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order ID format"})
		return
	}

	if err := h.orderService.DeleteOrder(c.Request.Context(), orderID, userID); err != nil {
		helpers.HandleError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// --- Mapper: Domain → HTTP Response ---

func toOrderResponse(order *entities.Order) *OrderResponse {
	return &OrderResponse{
		ID:          order.ID.String(),
		Title:       order.Title,
		Status:      string(order.Status),
		Description: order.Description,
		ProjectID:   order.ProjectID.String(),
	}
}
