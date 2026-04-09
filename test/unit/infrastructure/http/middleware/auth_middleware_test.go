package middleware_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"ductifact/internal/application/ports"
	"ductifact/internal/infrastructure/adapters/inbound/http/helpers"
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
	blacklist := &mocks.MockTokenBlacklist{}
	return setupRouterWithBlacklist(tokenProvider, blacklist)
}

// setupRouterWithBlacklist creates a test router with AuthMiddleware and a custom blacklist.
func setupRouterWithBlacklist(tokenProvider ports.TokenProvider, blacklist ports.TokenBlacklist) *gin.Engine {
	r := gin.New()

	r.GET("/protected", middleware.AuthMiddleware(tokenProvider, blacklist), func(c *gin.Context) {
		userID := helpers.MustGetUserID(c)
		if c.IsAborted() {
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

// --- Tests for MustGetUserID ---

func TestMustGetUserID_WithoutMiddleware_Returns401(t *testing.T) {
	// Create a router WITHOUT the middleware — simulates calling
	// MustGetUserID on a route that isn't protected.
	r := gin.New()

	r.GET("/unprotected", func(c *gin.Context) {
		_ = helpers.MustGetUserID(c)
		if c.IsAborted() {
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
	assert.Equal(t, "unauthorized", body["error"])
}

func TestMustGetUserID_WithWrongType_Returns401(t *testing.T) {
	// Simulate someone storing a non-UUID value with the same key.
	r := gin.New()

	r.GET("/bad-context", func(c *gin.Context) {
		c.Set(string(middleware.UserIDKey), "not-a-uuid") // wrong type

		_ = helpers.MustGetUserID(c)
		if c.IsAborted() {
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "should not reach here"})
	})

	req := httptest.NewRequest(http.MethodGet, "/bad-context", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_BlacklistedToken_Returns401(t *testing.T) {
	// ARRANGE
	userID := uuid.New()
	mockProvider := &mocks.MockTokenProvider{
		ValidateTokenFn: func(tokenString string) (*ports.TokenClaims, error) {
			return &ports.TokenClaims{UserID: userID, Email: "juan@example.com"}, nil
		},
	}
	blacklist := &mocks.MockTokenBlacklist{
		IsBlacklistedFn: func(token string) bool {
			return token == "revoked-access-token"
		},
	}

	router := setupRouterWithBlacklist(mockProvider, blacklist)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer revoked-access-token")
	w := httptest.NewRecorder()

	// ACT
	router.ServeHTTP(w, req)

	// ASSERT
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	assert.Equal(t, "token has been revoked", body["error"])
}
