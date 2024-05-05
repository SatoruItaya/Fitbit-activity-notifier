package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Interface for the mock function
type StepsByDateRangeFunc func(ctx context.Context, access_token string, startDate, endDate string) (map[string][]map[string]string, error)

// Mock function
func mockGetStepsByDateRange(ctx context.Context, access_token string, startDate, endDate string) (map[string][]map[string]string, error) {
	// Return dummy data for testing
	mockData := map[string][]map[string]string{
		"activities-steps": {
			{"dateTime": "2024-03-01", "value": "1000"},
			{"dateTime": "2024-03-02", "value": "1500"},
		},
	}
	return mockData, nil
}

func TestGetLifetimeStepsHistory(t *testing.T) {
	// Test input data
	access_token := "test_token"
	today := time.Date(2024, time.March, 3, 0, 0, 0, 0, time.Local)

	// Register the mock function
	var getStepsByDateRange StepsByDateRangeFunc = mockGetStepsByDateRange

	// Call the function under test
	stepsHistory, err := getLifetimeStepsHistory(context.Background(), access_token, today, getStepsByDateRange)

	// Verify no error occurred
	assert.NoError(t, err)

	// Verify correct step history is obtained
	expected := map[time.Time]int{
		time.Date(2024, time.March, 1, 0, 0, 0, 0, time.Local): 1000,
		time.Date(2024, time.March, 2, 0, 0, 0, 0, time.Local): 1500,
	}
	assert.Equal(t, expected, stepsHistory)
}

func TestGenerateStepsReport(t *testing.T) {
	lifetimeStepsData := map[time.Time]int{
		time.Date(2023, time.December, 15, 0, 0, 0, 0, time.Local): 1000,
		time.Date(2023, time.December, 16, 0, 0, 0, 0, time.Local): 1500,
		time.Date(2023, time.December, 17, 0, 0, 0, 0, time.Local): 2000,
		time.Date(2023, time.December, 18, 0, 0, 0, 0, time.Local): 2500,
		time.Date(2023, time.December, 15, 0, 0, 0, 0, time.Local): 3000,
		time.Date(2023, time.December, 19, 0, 0, 0, 0, time.Local): 4000,
		time.Date(2023, time.December, 20, 0, 0, 0, 0, time.Local): 5000,
		time.Date(2023, time.December, 21, 0, 0, 0, 0, time.Local): 6000,
		time.Date(2023, time.December, 22, 0, 0, 0, 0, time.Local): 7000,
		time.Date(2023, time.December, 23, 0, 0, 0, 0, time.Local): 8000,
		time.Date(2023, time.December, 24, 0, 0, 0, 0, time.Local): 9000,
		time.Date(2023, time.December, 25, 0, 0, 0, 0, time.Local): 10000,
		time.Date(2023, time.December, 26, 0, 0, 0, 0, time.Local): 17777,
		time.Date(2023, time.December, 27, 0, 0, 0, 0, time.Local): 10000,
		time.Date(2023, time.December, 28, 0, 0, 0, 0, time.Local): 10000,
		time.Date(2023, time.December, 29, 0, 0, 0, 0, time.Local): 10000,
		time.Date(2023, time.December, 30, 0, 0, 0, 0, time.Local): 10000,
		time.Date(2023, time.December, 31, 0, 0, 0, 0, time.Local): 23456,
		// ---yearly---
		time.Date(2024, time.January, 1, 0, 0, 0, 0, time.Local): 15000,
		time.Date(2024, time.January, 2, 0, 0, 0, 0, time.Local): 1000,
		time.Date(2024, time.January, 3, 0, 0, 0, 0, time.Local): 18999,
		time.Date(2024, time.January, 4, 0, 0, 0, 0, time.Local): 1500,
		time.Date(2024, time.January, 5, 0, 0, 0, 0, time.Local): 1000,
		time.Date(2024, time.January, 6, 0, 0, 0, 0, time.Local): 16666,
		// ---weekly---
		time.Date(2024, time.January, 7, 0, 0, 0, 0, time.Local):  1000,
		time.Date(2024, time.January, 8, 0, 0, 0, 0, time.Local):  2000,
		time.Date(2024, time.January, 9, 0, 0, 0, 0, time.Local):  1000,
		time.Date(2024, time.January, 10, 0, 0, 0, 0, time.Local): 18998,
		time.Date(2024, time.January, 11, 0, 0, 0, 0, time.Local): 1000,
		time.Date(2024, time.January, 12, 0, 0, 0, 0, time.Local): 1000,
		time.Date(2024, time.January, 13, 0, 0, 0, 0, time.Local): 19000,
	}

	expected := `
======================
Weekly Report

1/7 Sun 1,000
1/8 Mon 1,000
1/9 Tue 1,000
1/10 Wed 18,998
1/11 Thu 1,000
1/12 Fri 1,000
1/13 Sat 19,000

Total: 42,998
Average: 6,143
======================
Top Records in This Year

19,000(1/13)
18,999(1/3)
18,998(1/10)
16,666(1/6)
15,000(1/1)
======================
Top Records in Lifetime

23,456(2023/12/31)
19,000(2024/1/13)
18,999(2024/1/3)
18,998(2024/1/10)
17,777(2023/12/26)
`

	today := time.Date(2024, time.January, 14, 0, 0, 0, 0, time.Local)
	actual := generateStepsReport(lifetimeStepsData, today)

	if expected != actual {
		t.Errorf("Expected %v, but got %v", expected, actual)
	}
}
