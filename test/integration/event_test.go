package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"event-service/test/helpers"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateEvent_Success(t *testing.T) {
	router := helpers.SetupTestRouter(t)

	// Test data
	startTime := time.Now().Add(time.Hour)
	endTime := startTime.Add(2 * time.Hour)
	organizerID := uuid.New()

	requestBody := map[string]interface{}{
		"title":        "Test Event",
		"description":  "Test Description",
		"location":     "Test Location",
		"start_time":   startTime.Format(time.RFC3339),
		"end_time":     endTime.Format(time.RFC3339),
		"organizer_id": organizerID.String(),
	}

	jsonBody, _ := json.Marshal(requestBody)

	// Create request
	req, err := http.NewRequest("POST", "/events", bytes.NewBuffer(jsonBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	// Record response
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotEmpty(t, response["id"])
	assert.Equal(t, "Test Event", response["title"])
	assert.Equal(t, "Test Description", response["description"])
	assert.Equal(t, "Test Location", response["location"])
	assert.Equal(t, organizerID.String(), response["organizer_id"])
}

func TestCreateEvent_InvalidDuration(t *testing.T) {
	router := helpers.SetupTestRouter(t)

	// Test data with end time before start time
	startTime := time.Now().Add(2 * time.Hour)
	endTime := startTime.Add(-1 * time.Hour) // End before start
	organizerID := uuid.New()

	requestBody := map[string]interface{}{
		"title":        "Test Event",
		"description":  "Test Description",
		"location":     "Test Location",
		"start_time":   startTime.Format(time.RFC3339),
		"end_time":     endTime.Format(time.RFC3339),
		"organizer_id": organizerID.String(),
	}

	jsonBody, _ := json.Marshal(requestBody)

	// Create request
	req, err := http.NewRequest("POST", "/events", bytes.NewBuffer(jsonBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	// Record response
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateEvent_MissingRequiredFields(t *testing.T) {
	router := helpers.SetupTestRouter(t)

	// Test data missing required fields
	requestBody := map[string]interface{}{
		"title":       "Test Event",
		"description": "Test Description",
		// Missing location, start_time, end_time, organizer_id
	}

	jsonBody, _ := json.Marshal(requestBody)

	// Create request
	req, err := http.NewRequest("POST", "/events", bytes.NewBuffer(jsonBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	// Record response
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetEventByID_Success(t *testing.T) {
	router := helpers.SetupTestRouter(t)

	// First create an event
	startTime := time.Now().Add(time.Hour)
	endTime := startTime.Add(2 * time.Hour)
	organizerID := uuid.New()

	createRequestBody := map[string]interface{}{
		"title":        "Test Event",
		"description":  "Test Description",
		"location":     "Test Location",
		"start_time":   startTime.Format(time.RFC3339),
		"end_time":     endTime.Format(time.RFC3339),
		"organizer_id": organizerID.String(),
	}

	createJsonBody, _ := json.Marshal(createRequestBody)

	// Create event
	createReq, err := http.NewRequest("POST", "/events", bytes.NewBuffer(createJsonBody))
	require.NoError(t, err)
	createReq.Header.Set("Content-Type", "application/json")

	createW := httptest.NewRecorder()
	router.ServeHTTP(createW, createReq)

	// Parse created event ID
	var createResponse map[string]interface{}
	err = json.Unmarshal(createW.Body.Bytes(), &createResponse)
	require.NoError(t, err)

	eventID := createResponse["id"].(string)

	// Now get the event by ID
	getReq, err := http.NewRequest("GET", fmt.Sprintf("/events/%s", eventID), nil)
	require.NoError(t, err)

	getW := httptest.NewRecorder()
	router.ServeHTTP(getW, getReq)

	// Assertions
	assert.Equal(t, http.StatusOK, getW.Code)

	var getResponse map[string]interface{}
	err = json.Unmarshal(getW.Body.Bytes(), &getResponse)
	require.NoError(t, err)

	assert.Equal(t, eventID, getResponse["id"])
	assert.Equal(t, "Test Event", getResponse["title"])
}

func TestGetEventByID_NotFound(t *testing.T) {
	router := helpers.SetupTestRouter(t)

	// Try to get a non-existent event
	nonExistentID := uuid.New().String()
	req, err := http.NewRequest("GET", fmt.Sprintf("/events/%s", nonExistentID), nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHealthCheck(t *testing.T) {
	router := helpers.SetupTestRouter(t)

	req, err := http.NewRequest("GET", "/health", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
}
