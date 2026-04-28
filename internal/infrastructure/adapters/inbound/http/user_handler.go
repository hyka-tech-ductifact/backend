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
	Name   *string `json:"name" binding:"omitempty"`
	Email  *string `json:"email" binding:"omitempty,email"`
	Locale *string `json:"locale" binding:"omitempty,oneof=en es"`
}

type UserResponse struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Email         string `json:"email"`
	Locale        string `json:"locale"`
	EmailVerified bool   `json:"email_verified"`
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

	user, err := h.userService.UpdateUser(c.Request.Context(), userID, req.Name, req.Email, req.Locale)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, toUserResponse(user))
}

// DeleteMe handles DELETE /users/me — permanently deletes the authenticated user's account.
// This is a GDPR-compliant hard deletion that cascades to all user data.
// Requires ?cascade=true if the user has associated clients.
func (h *UserHandler) DeleteMe(c *gin.Context) {
	userID := helpers.MustGetUserID(c)
	if c.IsAborted() {
		return
	}

	cascade := c.Query("cascade") == "true"

	if err := h.userService.DeleteUser(c.Request.Context(), userID, cascade); err != nil {
		helpers.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "account deleted successfully"})
}

// --- Mapper: Domain → HTTP Response ---

func toUserResponse(user *entities.User) *UserResponse {
	return &UserResponse{
		ID:            user.ID.String(),
		Name:          user.Name,
		Email:         user.Email,
		Locale:        user.Locale,
		EmailVerified: user.IsEmailVerified(),
	}
}
