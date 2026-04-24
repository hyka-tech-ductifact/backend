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

func setupLivezRouter(healthMock *mocks.MockHealthChecker, storageMock *mocks.MockFileStorage) *gin.Engine {
	r := gin.New()
	h := handler.NewHealthHandler(healthMock, storageMock, time.Now(), config.ContractVersion, "test-version", "abc1234")
	r.GET("/healthz", h.Healthz)
	return r
}

func setupReadyzRouter(healthMock *mocks.MockHealthChecker, storageMock *mocks.MockFileStorage) *gin.Engine {
	r := gin.New()
	h := handler.NewHealthHandler(healthMock, storageMock, time.Now(), config.ContractVersion, "test-version", "abc1234")
	r.GET("/readyz", h.Readyz)
	return r
}

func TestHealthHandler_Livez(t *testing.T) {
	healthMock := &mocks.MockHealthChecker{}
	storageMock := &mocks.MockFileStorage{}

	router := setupLivezRouter(healthMock, storageMock)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"alive"`)
}

func TestHealthHandler_Readyz_Ready(t *testing.T) {
	healthMock := &mocks.MockHealthChecker{
		PingFn: func(ctx context.Context) error {
			return nil
		},
	}
	storageMock := &mocks.MockFileStorage{
		PingFn: func(ctx context.Context) error {
			return nil
		},
	}

	router := setupReadyzRouter(healthMock, storageMock)
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"ready"`)
	assert.Contains(t, w.Body.String(), `"database":"connected"`)
	assert.Contains(t, w.Body.String(), `"storage":"connected"`)
	assert.Contains(t, w.Body.String(), `"version":"test-version"`)
	assert.Contains(t, w.Body.String(), `"commit":"abc1234"`)
	assert.Contains(t, w.Body.String(), `"uptime"`)
	assert.Contains(t, w.Body.String(), fmt.Sprintf(`"contract_version":"%s"`, config.ContractVersion))
}

func TestHealthHandler_Readyz_DatabaseDown(t *testing.T) {
	healthMock := &mocks.MockHealthChecker{
		PingFn: func(ctx context.Context) error {
			return errors.New("connection refused")
		},
	}
	storageMock := &mocks.MockFileStorage{
		PingFn: func(ctx context.Context) error {
			return nil
		},
	}

	router := setupReadyzRouter(healthMock, storageMock)
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"not_ready"`)
	assert.Contains(t, w.Body.String(), `"database":"disconnected"`)
	assert.Contains(t, w.Body.String(), `"storage":"connected"`)
	assert.Contains(t, w.Body.String(), `"errors"`)
	assert.Contains(t, w.Body.String(), `database: connection refused`)
}

func TestHealthHandler_Readyz_StorageDown(t *testing.T) {
	healthMock := &mocks.MockHealthChecker{
		PingFn: func(ctx context.Context) error {
			return nil
		},
	}
	storageMock := &mocks.MockFileStorage{
		PingFn: func(ctx context.Context) error {
			return errors.New("minio unreachable")
		},
	}

	router := setupReadyzRouter(healthMock, storageMock)
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"not_ready"`)
	assert.Contains(t, w.Body.String(), `"database":"connected"`)
	assert.Contains(t, w.Body.String(), `"storage":"disconnected"`)
	assert.Contains(t, w.Body.String(), `storage: minio unreachable`)
}

func TestHealthHandler_Readyz_AllDown(t *testing.T) {
	healthMock := &mocks.MockHealthChecker{
		PingFn: func(ctx context.Context) error {
			return errors.New("connection refused")
		},
	}
	storageMock := &mocks.MockFileStorage{
		PingFn: func(ctx context.Context) error {
			return errors.New("minio unreachable")
		},
	}

	router := setupReadyzRouter(healthMock, storageMock)
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"not_ready"`)
	assert.Contains(t, w.Body.String(), `"database":"disconnected"`)
	assert.Contains(t, w.Body.String(), `"storage":"disconnected"`)
}
