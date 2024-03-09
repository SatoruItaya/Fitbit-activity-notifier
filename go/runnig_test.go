package main

import (
	"testing"
	"time"
)

func TestExtractRunningLog(t *testing.T) {
	var activityList []interface{}
	today := time.Date(2024, time.March, 13, 23, 59, 59, 999, time.UTC)

	startTime2024Running1String := "2024-03-04T20:09:59.000+09:00"
	startTime2024Running2String := "2024-03-12T20:09:59.000+09:00"
	startTime2024Running1Time, _ := time.Parse("2006-01-02T15:04:05.000-07:00", startTime2024Running1String)
	startTime2024Running2Time, _ := time.Parse("2006-01-02T15:04:05.000-07:00", startTime2024Running2String)

	distance2024Running1Float := 3.502289
	distance2024Running2Float := 2.603289

	//success
	activityList = append(activityList,
		map[string]interface{}{"activityName": "Run", "startTime": startTime2024Running1String, "distance": distance2024Running1Float},
		map[string]interface{}{"activityName": "Run", "startTime": startTime2024Running2String, "distance": distance2024Running2Float},
		map[string]interface{}{"activityName": "Run", "startTime": "2023-03-12T20:09:59.000+09:00", "distance": 2.603289},
		map[string]interface{}{"activityName": "Walk", "startTime": "2024-03-12T20:09:59.000+09:00", "distance": "false"},
	)

	yearlyRunningLog, err := extractRunningLog(activityList, today)
	if err != nil {
		t.Errorf("Error in extractRunningLog: %v", err)
	}

	if len(yearlyRunningLog) != 2 {
		t.Errorf("Expected 2 elements, but got %v element(s)", len(yearlyRunningLog))
	}

	if yearlyRunningLog[startTime2024Running1Time] != distance2024Running1Float {
		t.Errorf("Expected %v, but got %v", distance2024Running1Float, yearlyRunningLog[startTime2024Running1Time])
	}

	if yearlyRunningLog[startTime2024Running2Time] != distance2024Running2Float {
		t.Errorf("Expected %v, but got %v", distance2024Running2Float, yearlyRunningLog[startTime2024Running2Time])
	}

	//missing activityName
	var activityListMissingActiveName []interface{}
	activityListMissingActiveName = append(activityListMissingActiveName,
		map[string]interface{}{"startTime": startTime2024Running1String, "distance": distance2024Running1Float},
	)
	_, err = extractRunningLog(activityListMissingActiveName, today)
	if err == nil {
		t.Errorf("Expected error for unable to extract activityName, but got nil")
	} else if err.Error() != UnableToExtractAcitivityName {
		t.Errorf("Expected %v, but got %T", UnableToExtractAcitivityName, err)
	}

	//missing startTime
	var activityListMissingStartTime []interface{}
	activityListMissingStartTime = append(activityListMissingStartTime,
		map[string]interface{}{"activityName": "Run", "distance": distance2024Running1Float},
	)
	_, err = extractRunningLog(activityListMissingStartTime, today)
	if err == nil {
		t.Errorf("Expected error for unable to extract startTime, but got nil")
	} else if err.Error() != UnableToExtractStartTime {
		t.Errorf("Expected %v, but got %T", UnableToExtractStartTime, err)
	}

	//parse error
	var activityListParseError []interface{}
	activityListParseError = append(activityListParseError,
		map[string]interface{}{"activityName": "Run", "startTime": "2024-03-32T20:09:59.000+09:00", "distance": distance2024Running1Float},
	)
	_, err = extractRunningLog(activityListParseError, today)
	if err == nil {
		t.Errorf("Expected error for parse, but got nil")
	} else {
		if _, ok := err.(*time.ParseError); !ok {
			t.Errorf("Expected time.ParseError, but got %T", err)
		}
	}

	//missing distance
	var activityListMissingDistance []interface{}
	activityListMissingDistance = append(activityListMissingDistance,
		map[string]interface{}{"activityName": "Run", "startTime": "2024-03-30T20:09:59.000+09:00"},
	)
	_, err = extractRunningLog(activityListMissingDistance, today)
	if err == nil {
		t.Errorf("Expected error for unable to extract distance, but got nil")
	} else if err.Error() != UnableToExtractDistance {
		t.Errorf("Expected %v, but got %T", UnableToExtractDistance, err)
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

	acutual := generateRunningReport(yearlyRunningLog, today)
	if expected != acutual {
		t.Errorf("Expected %v, but got %v", expected, acutual)
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

	acutual = generateRunningReport(yearlyRunningLog, today)
	if expected != acutual {
		t.Errorf("Expected %v, but got %v", expected, acutual)
	}

}
