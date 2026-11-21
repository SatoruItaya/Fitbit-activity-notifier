package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func setHealthAPIServer(t *testing.T, handler http.HandlerFunc) {
	server := httptest.NewServer(handler)
	originalBaseURL := healthAPIBaseURL
	healthAPIBaseURL = server.URL
	t.Cleanup(func() {
		healthAPIBaseURL = originalBaseURL
		server.Close()
	})
}

func TestGetStepsByDateRange(t *testing.T) {
	setHealthAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/steps/dataPoints:dailyRollUp", r.URL.Path)
		assert.Equal(t, "Bearer test_token", r.Header.Get("Authorization"))

		var req dailyRollUpRequest
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, CivilDate{Year: 2026, Month: 2, Day: 25}, req.Range.Start.Date)
		assert.Equal(t, CivilDate{Year: 2026, Month: 2, Day: 28}, req.Range.End.Date)
		assert.Equal(t, 1, req.WindowSizeDays)

		// Response is reverse-chronological and steps is omitted for days without data
		_, _ = w.Write([]byte(`{
  "rollupDataPoints": [
    {"civilStartTime": {"date": {"year": 2026, "month": 2, "day": 27}, "time": {}}, "civilEndTime": {"date": {"year": 2026, "month": 2, "day": 28}, "time": {}}, "steps": {"countSum": "3822"}},
    {"civilStartTime": {"date": {"year": 2026, "month": 2, "day": 26}, "time": {}}, "civilEndTime": {"date": {"year": 2026, "month": 2, "day": 27}, "time": {}}},
    {"civilStartTime": {"date": {"year": 2026, "month": 2, "day": 25}, "time": {}}, "civilEndTime": {"date": {"year": 2026, "month": 2, "day": 26}, "time": {}}, "steps": {"countSum": "12345"}}
  ]
}`))
	})

	start := time.Date(2026, time.February, 25, 0, 0, 0, 0, time.Local)
	end := time.Date(2026, time.February, 28, 0, 0, 0, 0, time.Local)
	actual, err := getStepsByDateRange(context.Background(), "test_token", start, end)
	assert.NoError(t, err)

	expected := map[time.Time]int{
		time.Date(2026, time.February, 25, 0, 0, 0, 0, time.Local): 12345,
		time.Date(2026, time.February, 26, 0, 0, 0, 0, time.Local): 0,
		time.Date(2026, time.February, 27, 0, 0, 0, 0, time.Local): 3822,
	}
	assert.Equal(t, expected, actual)
}

func TestGetStepsByDateRangeError(t *testing.T) {
	setHealthAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error": {"code": 403, "status": "PERMISSION_DENIED"}}`))
	})

	day := time.Date(2026, time.February, 25, 0, 0, 0, 0, time.Local)
	_, err := getStepsByDateRange(context.Background(), "test_token", day, day.AddDate(0, 0, 1))
	assert.ErrorContains(t, err, "403")
}

func TestGetExerciseList(t *testing.T) {
	requestCount := 0
	setHealthAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/exercise/dataPoints", r.URL.Path)
		assert.Equal(t, `exercise.interval.civil_start_time >= "2026-01-01"`, r.URL.Query().Get("filter"))
		assert.Equal(t, "25", r.URL.Query().Get("pageSize"))

		requestCount++
		switch r.URL.Query().Get("pageToken") {
		case "":
			_, _ = w.Write([]byte(`{
  "dataPoints": [
    {"name": "users/1/dataTypes/exercise/dataPoints/1", "exercise": {"interval": {"startTime": "2026-04-20T08:00:00Z", "startUtcOffset": "32400s"}, "exerciseType": "RUNNING", "metricsSummary": {"distanceMillimeters": 5000000}, "displayName": "Run"}}
  ],
  "nextPageToken": "next"
}`))
		case "next":
			_, _ = w.Write([]byte(`{
  "dataPoints": [
    {"name": "users/1/dataTypes/exercise/dataPoints/2", "exercise": {"interval": {"startTime": "2026-02-23T13:10:00Z", "startUtcOffset": "32400s"}, "exerciseType": "WALKING", "metricsSummary": {"distanceMillimeters": 1609344}, "displayName": "Walk"}}
  ],
  "nextPageToken": ""
}`))
		default:
			t.Errorf("Unexpected pageToken: %v", r.URL.Query().Get("pageToken"))
		}
	})

	today := time.Date(2026, time.November, 14, 12, 0, 0, 0, time.UTC)
	exerciseList, err := getExerciseList(context.Background(), "test_token", today)
	assert.NoError(t, err)
	assert.Equal(t, 2, requestCount)
	assert.Len(t, exerciseList, 2)

	yearlyRunningLog, err := extractRunningLog(exerciseList, today)
	assert.NoError(t, err)
	assert.Len(t, yearlyRunningLog, 1)
	for k, v := range yearlyRunningLog {
		assert.True(t, k.Equal(time.Date(2026, time.April, 20, 8, 0, 0, 0, time.UTC)))
		assert.Equal(t, 5.0, v)
	}
}
