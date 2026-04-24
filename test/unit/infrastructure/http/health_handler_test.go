package http_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ductifact/internal/config"
	handler "ductifact/internal/infrastructure/adapters/inbound/http"
	"ductifact/test/unit/mocks"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupLivezRouter(mock *mocks.MockHealthChecker) *gin.Engine {
	r := gin.New()
	h := handler.NewHealthHandler(mock, time.Now(), config.ContractVersion)
	r.GET("/healthz", h.Healthz)
	return r
}

func setupReadyzRouter(mock *mocks.MockHealthChecker) *gin.Engine {
	r := gin.New()
	h := handler.NewHealthHandler(mock, time.Now(), config.ContractVersion)
	r.GET("/readyz", h.Readyz)
	return r
}

func TestHealthHandler_Livez(t *testing.T) {
	mock := &mocks.MockHealthChecker{}

	router := setupLivezRouter(mock)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"alive"`)
}

func TestHealthHandler_Readyz_Ready(t *testing.T) {
	mock := &mocks.MockHealthChecker{
		PingFn: func(ctx context.Context) error {
			return nil
		},
	}

	router := setupReadyzRouter(mock)
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"ready"`)
	assert.Contains(t, w.Body.String(), `"database":"connected"`)
	assert.Contains(t, w.Body.String(), `"uptime"`)
	assert.Contains(t, w.Body.String(), fmt.Sprintf(`"contract_version":"%s"`, config.ContractVersion))
}

func TestHealthHandler_Readyz_NotReady(t *testing.T) {
	mock := &mocks.MockHealthChecker{
		PingFn: func(ctx context.Context) error {
			return errors.New("connection refused")
		},
	}

	router := setupReadyzRouter(mock)
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"not_ready"`)
	assert.Contains(t, w.Body.String(), `"database":"disconnected"`)
	assert.Contains(t, w.Body.String(), `"error":"connection refused"`)
	assert.Contains(t, w.Body.String(), fmt.Sprintf(`"contract_version":"%s"`, config.ContractVersion))
}
