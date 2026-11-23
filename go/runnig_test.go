package main

import (
	"testing"
	"time"
)

func newExercise(exerciseType string, startTime string, startUtcOffset string, distanceMillimeters float64) Exercise {
	var e Exercise
	e.ExerciseType = exerciseType
	e.Interval.StartTime = startTime
	e.Interval.StartUtcOffset = startUtcOffset
	e.MetricsSummary.DistanceMillimeters = distanceMillimeters
	return e
}

func TestExtractRunningLog(t *testing.T) {
	today := time.Date(2024, time.March, 13, 23, 59, 59, 999, time.UTC)
	jst := time.FixedZone("", 9*60*60)

	running1Time := time.Date(2024, time.March, 4, 20, 9, 59, 0, jst)
	// 2024-03-12 08:09:59 JST is 2024-03-11 in UTC
	running2Time := time.Date(2024, time.March, 12, 8, 9, 59, 0, jst)

	//success
	exerciseList := []Exercise{
		newExercise("RUNNING", "2024-03-04T11:09:59Z", "32400s", 3502289),
		newExercise("RUNNING", "2024-03-11T23:09:59Z", "32400s", 2603289),
		newExercise("RUNNING", "2023-03-12T11:09:59Z", "32400s", 2603289),
		// 2024-01-01 08:00 JST is 2023-12-31 in UTC, but it is in this year
		newExercise("RUNNING", "2023-12-31T23:00:00Z", "32400s", 1000000),
		newExercise("WALKING", "2024-03-12T11:09:59Z", "32400s", 0),
	}

	yearlyRunningLog, err := extractRunningLog(exerciseList, today)
	if err != nil {
		t.Errorf("Error in extractRunningLog: %v", err)
	}

	if len(yearlyRunningLog) != 3 {
		t.Errorf("Expected 3 elements, but got %v element(s)", len(yearlyRunningLog))
	}

	for k, v := range yearlyRunningLog {
		switch {
		case k.Equal(running1Time):
			if v != 3.502289 {
				t.Errorf("Expected %v, but got %v", 3.502289, v)
			}
			if k.Format(YEARLY_REPORT_DATE_FORMAT) != "3/4" {
				t.Errorf("Expected local date 3/4, but got %v", k.Format(YEARLY_REPORT_DATE_FORMAT))
			}
		case k.Equal(running2Time):
			if v != 2.603289 {
				t.Errorf("Expected %v, but got %v", 2.603289, v)
			}
			if k.Format(YEARLY_REPORT_DATE_FORMAT) != "3/12" {
				t.Errorf("Expected local date 3/12, but got %v", k.Format(YEARLY_REPORT_DATE_FORMAT))
			}
		case v == 1:
		default:
			t.Errorf("Unexpected element %v: %v", k, v)
		}
	}

	//missing exerciseType
	_, err = extractRunningLog([]Exercise{newExercise("", "2024-03-04T11:09:59Z", "32400s", 3502289)}, today)
	if err == nil {
		t.Errorf("Expected error for unable to extract exerciseType, but got nil")
	} else if err.Error() != UnableToExtractExerciseType {
		t.Errorf("Expected %v, but got %v", UnableToExtractExerciseType, err)
	}

	//missing startTime
	_, err = extractRunningLog([]Exercise{newExercise("RUNNING", "", "32400s", 3502289)}, today)
	if err == nil {
		t.Errorf("Expected error for unable to extract startTime, but got nil")
	} else if err.Error() != UnableToExtractStartTime {
		t.Errorf("Expected %v, but got %v", UnableToExtractStartTime, err)
	}

	//parse error
	_, err = extractRunningLog([]Exercise{newExercise("RUNNING", "2024-03-32T11:09:59Z", "32400s", 3502289)}, today)
	if err == nil {
		t.Errorf("Expected error for parse, but got nil")
	} else {
		if _, ok := err.(*time.ParseError); !ok {
			t.Errorf("Expected time.ParseError, but got %T", err)
		}
	}

	//missing distance is treated as 0
	yearlyRunningLog, err = extractRunningLog([]Exercise{newExercise("RUNNING", "2024-03-04T11:09:59Z", "32400s", 0)}, today)
	if err != nil {
		t.Errorf("Error in extractRunningLog: %v", err)
	}
	for _, v := range yearlyRunningLog {
		if v != 0 {
			t.Errorf("Expected 0, but got %v", v)
		}
	}
}

func TestGenerateRunningReport(t *testing.T) {
	today := time.Date(2024, time.March, 9, 23, 59, 59, 999, time.UTC)

	// There are runnning activities foa a week
	yearlyRunningLog := map[time.Time]float64{
		time.Date(2024, time.March, 8, 23, 59, 59, 999, time.UTC):   3.08,
		time.Date(2024, time.March, 7, 23, 59, 59, 999, time.UTC):   5.08,
		time.Date(2024, time.January, 2, 23, 59, 59, 999, time.UTC): 10.08,
	}

	expected := `
======================
Running Report
3/7 Thu 5.08km
3/8 Fri 3.08km

Weekly Distance: 8.16km
Yearly Distance: 18.24km`

	actual := generateRunningReport(yearlyRunningLog, today)
	if expected != actual {
		t.Errorf("Expected %v, but got %v", expected, actual)
	}

	// Runs in the morning of the first day of the week are included,
	// and runs on the day before the week and on today are excluded
	today = time.Date(2025, time.January, 26, 12, 0, 0, 000, time.UTC)
	yearlyRunningLog = map[time.Time]float64{
		time.Date(2025, time.January, 18, 23, 0, 0, 0, time.UTC):    1.01,
		time.Date(2025, time.January, 19, 8, 0, 0, 0, time.UTC):     10.06,
		time.Date(2025, time.January, 19, 12, 59, 59, 0, time.UTC):  5.03,
		time.Date(2025, time.January, 26, 8, 0, 0, 0, time.UTC):     2.02,
		time.Date(2025, time.January, 2, 23, 59, 59, 999, time.UTC): 19.46,
	}

	expected = `
======================
Running Report
1/19 Sun 10.06km
1/19 Sun 5.03km

Weekly Distance: 15.09km
Yearly Distance: 37.58km`

	actual = generateRunningReport(yearlyRunningLog, today)
	if expected != actual {
		t.Errorf("Expected %v, but got %v", expected, actual)
	}

	// Lambda runs in UTC (Sunday 12:00 JST is Sunday 03:00 UTC),
	// and start times are in the local time of each run (JST)
	jst := time.FixedZone("", 9*60*60)
	today = time.Date(2026, time.November, 22, 3, 0, 0, 0, time.UTC)
	yearlyRunningLog = map[time.Time]float64{
		time.Date(2026, time.November, 14, 23, 0, 0, 0, jst): 1.01,
		time.Date(2026, time.November, 15, 7, 0, 0, 0, jst):  5.5,
		time.Date(2026, time.November, 21, 23, 0, 0, 0, jst): 3.3,
		time.Date(2026, time.November, 22, 7, 0, 0, 0, jst):  2.02,
	}

	expected = `
======================
Running Report
11/15 Sun 5.5km
11/21 Sat 3.3km

Weekly Distance: 8.8km
Yearly Distance: 11.83km`

	actual = generateRunningReport(yearlyRunningLog, today)
	if expected != actual {
		t.Errorf("Expected %v, but got %v", expected, actual)
	}

	// There are no runnning activities foa a week
	yearlyRunningLog = map[time.Time]float64{
		time.Date(2024, time.January, 2, 23, 59, 59, 999, time.UTC): 10.08,
	}
	expected = `
======================
Running Report

Weekly Distance: 0km
Yearly Distance: 10.08km`

	actual = generateRunningReport(yearlyRunningLog, today)
	if expected != actual {
		t.Errorf("Expected %v, but got %v", expected, actual)
	}

}
