package middleware_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"ductifact/internal/application/ports"
	"ductifact/internal/infrastructure/adapters/inbound/http/middleware"
	"ductifact/test/unit/mocks"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// init sets Gin to test mode to suppress debug output during tests.
func init() {
	gin.SetMode(gin.TestMode)
}

// setupRouter creates a test router with AuthMiddleware applied to a test endpoint.
// The handler behind the middleware writes the userID from context into the response body,
// so we can verify that the middleware set it correctly.
func setupRouter(tokenProvider ports.TokenProvider) *gin.Engine {
	r := gin.New()

	r.GET("/protected", middleware.AuthMiddleware(tokenProvider), func(c *gin.Context) {
		userID, err := middleware.GetUserIDFromContext(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"user_id": userID.String()})
	})

	return r
}

// --- Tests ---

func TestAuthMiddleware_ValidToken_PassesThrough(t *testing.T) {
	expectedUserID := uuid.New()

	mockToken := &mocks.MockTokenProvider{
		ValidateTokenFn: func(tokenString string) (*ports.TokenClaims, error) {
			return &ports.TokenClaims{
				UserID: expectedUserID,
				Email:  "test@example.com",
			}, nil
		},
	}

	router := setupRouter(mockToken)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer valid-token-here")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, expectedUserID.String(), body["user_id"])
}

func TestAuthMiddleware_MissingAuthorizationHeader_Returns401(t *testing.T) {
	mockToken := &mocks.MockTokenProvider{}

	router := setupRouter(mockToken)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	// No Authorization header
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "authorization header required", body["error"])
}

func TestAuthMiddleware_MalformedHeader_NoBearerPrefix_Returns401(t *testing.T) {
	mockToken := &mocks.MockTokenProvider{}

	router := setupRouter(mockToken)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Token some-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "authorization header must be: Bearer <token>", body["error"])
}

func TestAuthMiddleware_MalformedHeader_BearerOnly_Returns401(t *testing.T) {
	mockToken := &mocks.MockTokenProvider{}

	router := setupRouter(mockToken)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_InvalidToken_Returns401(t *testing.T) {
	mockToken := &mocks.MockTokenProvider{
		ValidateTokenFn: func(tokenString string) (*ports.TokenClaims, error) {
			return nil, errors.New("invalid or expired token")
		},
	}

	router := setupRouter(mockToken)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "invalid or expired token", body["error"])
}

func TestAuthMiddleware_ExpiredToken_Returns401(t *testing.T) {
	mockToken := &mocks.MockTokenProvider{
		ValidateTokenFn: func(tokenString string) (*ports.TokenClaims, error) {
			return nil, errors.New("invalid or expired token")
		},
	}

	router := setupRouter(mockToken)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer expired-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- Tests for GetUserIDFromContext ---

func TestGetUserIDFromContext_WithoutMiddleware_ReturnsError(t *testing.T) {
	// Create a router WITHOUT the middleware — simulates calling
	// GetUserIDFromContext on a route that isn't protected.
	r := gin.New()

	r.GET("/unprotected", func(c *gin.Context) {
		_, err := middleware.GetUserIDFromContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "should not reach here"})
	})

	req := httptest.NewRequest(http.MethodGet, "/unprotected", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "user ID not found in context", body["error"])
}

func TestGetUserIDFromContext_WithWrongType_ReturnsError(t *testing.T) {
	// Simulate someone storing a non-UUID value with the same key.
	r := gin.New()

	r.GET("/bad-context", func(c *gin.Context) {
		c.Set(string(middleware.UserIDKey), "not-a-uuid") // wrong type

		_, err := middleware.GetUserIDFromContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "should not reach here"})
	})

	req := httptest.NewRequest(http.MethodGet, "/bad-context", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
