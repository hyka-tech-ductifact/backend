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
	h := handler.NewHealthHandler(
		healthMock,
		storageMock,
		&mocks.MockEmailSender{},
		time.Now(),
		config.ContractVersion,
		"test-version",
		"abc1234",
		"debug",
	)
	r.GET("/healthz", h.Healthz)
	return r
}

func setupReadyzRouter(healthMock *mocks.MockHealthChecker, storageMock *mocks.MockFileStorage) *gin.Engine {
	r := gin.New()
	h := handler.NewHealthHandler(
		healthMock,
		storageMock,
		&mocks.MockEmailSender{},
		time.Now(),
		config.ContractVersion,
		"test-version",
		"abc1234",
		"debug",
	)
	r.GET("/readyz", h.Readyz)
	return r
}

func setupReadyzRouterWithEmail(
	healthMock *mocks.MockHealthChecker,
	storageMock *mocks.MockFileStorage,
	emailMock *mocks.MockEmailSender,
) *gin.Engine {
	r := gin.New()
	h := handler.NewHealthHandler(
		healthMock,
		storageMock,
		emailMock,
		time.Now(),
		config.ContractVersion,
		"test-version",
		"abc1234",
		"debug",
	)
	r.GET("/readyz", h.Readyz)
	return r
}

func setupReadyzRouterProd(
	healthMock *mocks.MockHealthChecker,
	storageMock *mocks.MockFileStorage,
	emailMock *mocks.MockEmailSender,
) *gin.Engine {
	r := gin.New()
	h := handler.NewHealthHandler(
		healthMock,
		storageMock,
		emailMock,
		time.Now(),
		config.ContractVersion,
		"test-version",
		"abc1234",
		"info",
	)
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
	assert.Contains(t, w.Body.String(), `"email":"connected"`)
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
	emailMock := &mocks.MockEmailSender{
		PingFn: func(ctx context.Context) error {
			return errors.New("smtp unreachable")
		},
	}

	router := setupReadyzRouterWithEmail(healthMock, storageMock, emailMock)
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"not_ready"`)
	assert.Contains(t, w.Body.String(), `"database":"disconnected"`)
	assert.Contains(t, w.Body.String(), `"storage":"disconnected"`)
	assert.Contains(t, w.Body.String(), `"email":"unavailable"`)
	assert.Contains(t, w.Body.String(), `"warnings"`)
}

func TestHealthHandler_Readyz_EmailDown_IsDegraded(t *testing.T) {
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
	emailMock := &mocks.MockEmailSender{
		PingFn: func(ctx context.Context) error {
			return errors.New("smtp unreachable")
		},
	}

	router := setupReadyzRouterWithEmail(healthMock, storageMock, emailMock)
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Email is non-critical: returns 200 with "degraded" status
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"degraded"`)
	assert.Contains(t, w.Body.String(), `"database":"connected"`)
	assert.Contains(t, w.Body.String(), `"storage":"connected"`)
	assert.Contains(t, w.Body.String(), `"email":"unavailable"`)
	assert.Contains(t, w.Body.String(), `"warnings"`)
	assert.Contains(t, w.Body.String(), `email: smtp unreachable`)
	// No errors array
	assert.NotContains(t, w.Body.String(), `"errors"`)
}

func TestHealthHandler_Readyz_NonDebug_SanitizesErrors(t *testing.T) {
	healthMock := &mocks.MockHealthChecker{
		PingFn: func(ctx context.Context) error {
			return errors.New("dial tcp 10.0.0.5:5432: connection refused")
		},
	}
	storageMock := &mocks.MockFileStorage{
		PingFn: func(ctx context.Context) error {
			return nil
		},
	}
	emailMock := &mocks.MockEmailSender{
		PingFn: func(ctx context.Context) error {
			return errors.New(
				"smtp ping smtp.sendgrid.net:587: dial tcp: lookup smtp.sendgrid.net on 127.0.0.11:53: no such host",
			)
		},
	}

	router := setupReadyzRouterProd(healthMock, storageMock, emailMock)
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"not_ready"`)
	// In prod, errors show generic messages, not internal details
	assert.Contains(t, w.Body.String(), `"database: unavailable"`)
	assert.NotContains(t, w.Body.String(), `10.0.0.5`)
	assert.NotContains(t, w.Body.String(), `dial tcp`)
	// Warnings also sanitized
	assert.Contains(t, w.Body.String(), `"email: unavailable"`)
	assert.NotContains(t, w.Body.String(), `smtp.sendgrid.net`)
	assert.NotContains(t, w.Body.String(), `127.0.0.11`)
}
