package http

import (
	"net/http"

	"ductifact/internal/application/usecases"
	"ductifact/internal/domain/entities"
	"ductifact/internal/infrastructure/adapters/inbound/http/helpers"

	"github.com/gin-gonic/gin"
)

// --- DTOs (HTTP-specific, not domain objects) ---

type UpdateUserRequest struct {
	Name  *string `json:"name" binding:"omitempty"`
	Email *string `json:"email" binding:"omitempty,email"`
}

type UserResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// --- Handler ---

type UserHandler struct {
	userService usecases.UserService
}

func NewUserHandler(userService usecases.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// GetMe handles GET /users/me — returns the authenticated user's profile.
func (h *UserHandler) GetMe(c *gin.Context) {
	userID := helpers.MustGetUserID(c)
	if c.IsAborted() {
		return
	}

	user, err := h.userService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, toUserResponse(user))
}

// UpdateMe handles PUT /users/me — updates the authenticated user's profile.
func (h *UserHandler) UpdateMe(c *gin.Context) {
	userID := helpers.MustGetUserID(c)
	if c.IsAborted() {
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.userService.UpdateUser(c.Request.Context(), userID, req.Name, req.Email)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, toUserResponse(user))
}

// --- Mapper: Domain → HTTP Response ---

func toUserResponse(user *entities.User) *UserResponse {
	return &UserResponse{
		ID:    user.ID.String(),
		Name:  user.Name,
		Email: user.Email,
	}
}
