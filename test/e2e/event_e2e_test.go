package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEventE2E_CompleteUserWorkflow tests a complete user workflow
// This is what makes E2E tests different - they test complete scenarios
func TestEventE2E_CompleteUserWorkflow(t *testing.T) {
	env := SetupTestEnvironment(t)
	defer env.Cleanup()

	// Step 1: Create multiple events for the same organizer
	organizerID := uuid.New()
	events := []map[string]interface{}{
		{
			"title":        "Morning Conference",
			"description":  "Tech conference in the morning",
			"location":     "Conference Center A",
			"start_time":   time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			"end_time":     time.Now().Add(26 * time.Hour).Format(time.RFC3339),
			"organizer_id": organizerID.String(),
		},
		{
			"title":        "Afternoon Workshop",
			"description":  "Hands-on workshop",
			"location":     "Workshop Room B",
			"start_time":   time.Now().Add(28 * time.Hour).Format(time.RFC3339),
			"end_time":     time.Now().Add(30 * time.Hour).Format(time.RFC3339),
			"organizer_id": organizerID.String(),
		},
	}

	var createdEventIDs []string

	// Create all events
	for _, eventData := range events {
		resp, err := http.Post(
			env.APIBaseURL+"/events",
			"application/json",
			bytes.NewBuffer(mustMarshal(t, eventData)),
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		resp.Body.Close()

		eventID := response["id"].(string)
		createdEventIDs = append(createdEventIDs, eventID)
	}

	// Step 2: Verify all events were created and are accessible
	for i, eventID := range createdEventIDs {
		resp, err := http.Get(env.APIBaseURL + "/events/" + eventID)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		resp.Body.Close()

		// Verify the event data matches what we created
		assert.Equal(t, events[i]["title"], response["title"])
		assert.Equal(t, events[i]["location"], response["location"])
		assert.Equal(t, organizerID.String(), response["organizer_id"])
	}

	// Step 3: Test system resilience - try to access non-existent event
	nonExistentID := uuid.New().String()
	resp, err := http.Get(env.APIBaseURL + "/events/" + nonExistentID)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()
}

// TestEventE2E_SystemHealthAndRecovery tests system health and recovery scenarios
func TestEventE2E_SystemHealthAndRecovery(t *testing.T) {
	env := SetupTestEnvironment(t)
	defer env.Cleanup()

	// Test 1: Health check endpoint
	resp, err := http.Get(env.APIBaseURL + "/health")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var healthResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&healthResponse)
	require.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, "healthy !!!!", healthResponse["status"])

	// Test 2: System handles malformed requests gracefully
	malformedRequests := []string{
		`{"invalid": "json"`,                // Invalid JSON
		`{"title": "Test"}`,                 // Missing required fields
		`{"title": "", "location": "Test"}`, // Empty required fields
	}

	for _, malformedReq := range malformedRequests {
		resp, err := http.Post(
			env.APIBaseURL+"/events",
			"application/json",
			bytes.NewBuffer([]byte(malformedReq)),
		)
		require.NoError(t, err)
		// Should return 400 Bad Request for malformed requests
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		resp.Body.Close()
	}
}

// TestEventE2E_DataConsistency tests data consistency across operations
func TestEventE2E_DataConsistency(t *testing.T) {
	env := SetupTestEnvironment(t)
	defer env.Cleanup()

	// Create an event
	startTime := time.Now().Add(time.Hour)
	endTime := startTime.Add(2 * time.Hour)
	organizerID := uuid.New()

	eventData := map[string]interface{}{
		"title":        "Consistency Test Event",
		"description":  "Testing data consistency",
		"location":     "Test Location",
		"start_time":   startTime.Format(time.RFC3339),
		"end_time":     endTime.Format(time.RFC3339),
		"organizer_id": organizerID.String(),
	}

	// Create event
	createResp, err := http.Post(
		env.APIBaseURL+"/events",
		"application/json",
		bytes.NewBuffer(mustMarshal(t, eventData)),
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, createResp.StatusCode)

	var createResponse map[string]interface{}
	err = json.NewDecoder(createResp.Body).Decode(&createResponse)
	require.NoError(t, err)
	createResp.Body.Close()

	eventID := createResponse["id"].(string)

	// Verify data consistency by retrieving the event multiple times
	for i := 0; i < 3; i++ {
		resp, err := http.Get(env.APIBaseURL + "/events/" + eventID)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		resp.Body.Close()

		// Data should be consistent across retrievals
		assert.Equal(t, "Consistency Test Event", response["title"])
		assert.Equal(t, "Testing data consistency", response["description"])
		assert.Equal(t, "Test Location", response["location"])
		assert.Equal(t, organizerID.String(), response["organizer_id"])
	}
}

// Helper function
func mustMarshal(t *testing.T, v interface{}) []byte {
	data, err := json.Marshal(v)
	require.NoError(t, err)
	return data
}
