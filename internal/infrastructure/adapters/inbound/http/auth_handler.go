package http

import (
	"net/http"
	"strings"

	"ductifact/internal/application/usecases"
	"ductifact/internal/infrastructure/adapters/inbound/http/helpers"

	"github.com/gin-gonic/gin"
)

// --- DTOs ---

type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type AuthResponse struct {
	User         UserResponse `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// --- Handler ---

type AuthHandler struct {
	authService usecases.AuthService
}

func NewAuthHandler(authService usecases.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := helpers.StrictBindJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, tokens, err := h.authService.Register(c.Request.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, AuthResponse{
		User:         *toUserResponse(user),
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := helpers.StrictBindJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, tokens, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		User:         *toUserResponse(user),
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := helpers.StrictBindJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tokens, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, TokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := helpers.StrictBindJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Extract the access token from the Authorization header
	authHeader := c.GetHeader("Authorization")
	accessToken := ""
	if parts := strings.SplitN(authHeader, " ", 2); len(parts) == 2 {
		accessToken = parts[1]
	}

	if err := h.authService.Logout(c.Request.Context(), accessToken, req.RefreshToken); err != nil {
		helpers.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}
