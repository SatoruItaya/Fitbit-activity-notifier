package main

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const EXERCISE_PAGE_SIZE = 25

type Exercise struct {
	Interval struct {
		StartTime      string `json:"startTime"`
		StartUtcOffset string `json:"startUtcOffset"`
	} `json:"interval"`
	ExerciseType   string `json:"exerciseType"`
	MetricsSummary struct {
		DistanceMillimeters float64 `json:"distanceMillimeters"`
	} `json:"metricsSummary"`
}

type exerciseListResponse struct {
	DataPoints []struct {
		Exercise Exercise `json:"exercise"`
	} `json:"dataPoints"`
	NextPageToken string `json:"nextPageToken"`
}

// getExerciseList returns exercises started in this year.
func getExerciseList(ctx context.Context, access_token string, today time.Time) ([]Exercise, error) {
	query := url.Values{}
	query.Set("filter", `exercise.interval.civil_start_time >= "`+strconv.Itoa(today.Year())+`-01-01"`)
	query.Set("pageSize", strconv.Itoa(EXERCISE_PAGE_SIZE))

	var exerciseList []Exercise

	for {
		apiUrl := healthAPIBaseURL + "/exercise/dataPoints?" + strings.ReplaceAll(query.Encode(), "+", "%20")

		var responseData exerciseListResponse
		if err := callHealthAPI(ctx, access_token, http.MethodGet, apiUrl, nil, &responseData); err != nil {
			return nil, err
		}

		for _, dataPoint := range responseData.DataPoints {
			exerciseList = append(exerciseList, dataPoint.Exercise)
		}

		if responseData.NextPageToken == "" {
			break
		}
		query.Set("pageToken", responseData.NextPageToken)
	}

	return exerciseList, nil
}

func extractRunningLog(exerciseList []Exercise, today time.Time) (map[time.Time]float64, error) {
	yearlyRunningLog := map[time.Time]float64{}

	for _, exercise := range exerciseList {
		if exercise.ExerciseType == "" {
			return nil, errors.New(UnableToExtractExerciseType)
		}

		if exercise.Interval.StartTime == "" {
			return nil, errors.New(UnableToExtractStartTime)
		}
		startTime, err := time.Parse(time.RFC3339, exercise.Interval.StartTime)
		if err != nil {
			return nil, err
		}

		// startTime is UTC, so convert it to the local time of the exercise
		if exercise.Interval.StartUtcOffset != "" {
			offset, err := time.ParseDuration(exercise.Interval.StartUtcOffset)
			if err != nil {
				return nil, err
			}
			startTime = startTime.In(time.FixedZone("", int(offset.Seconds())))
		}

		if exercise.ExerciseType == "RUNNING" {
			// millimeters to kilometers
			// distanceMillimeters is omitted when it is 0 (e.g. manually logged runs without distance)
			distance := exercise.MetricsSummary.DistanceMillimeters / 1000000

			if startTime.Year() == today.Year() {
				yearlyRunningLog[startTime] = distance
			}
		}
	}

	return yearlyRunningLog, nil
}

func generateRunningReport(yearlyRunningLog map[time.Time]float64, today time.Time) string {
	var keys []time.Time
	for key := range yearlyRunningLog {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].Before(keys[j])
	})

	yearlyDistance := 0.0
	weeklyDistance := 0.0
	// Weekly report covers the 7 days before today (same as steps report).
	// Compare by date because each start time is in the local time of the exercise.
	weekStartDate := time.Date(today.Year(), today.Month(), today.Day()-7, 0, 0, 0, 0, time.UTC)
	weekEndDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	report := "\n" + SEPARATOR + "Running Report\n"

	for _, k := range keys {
		date := time.Date(k.Year(), k.Month(), k.Day(), 0, 0, 0, 0, time.UTC)
		if !date.Before(weekStartDate) && date.Before(weekEndDate) {
			report += k.Format(YEARLY_REPORT_DATE_FORMAT) + " " + k.Format(DAY_OF_WEEK_FORMAT) + " " + strconv.FormatFloat(roundToDecimal(yearlyRunningLog[k]), 'f', -1, 64) + "km\n"
			weeklyDistance += yearlyRunningLog[k]
		}
		yearlyDistance += yearlyRunningLog[k]
	}

	report += "\n"
	report += "Weekly Distance: " + strconv.FormatFloat(roundToDecimal(weeklyDistance), 'f', -1, 64) + "km\n"
	report += "Yearly Distance: " + strconv.FormatFloat(roundToDecimal(yearlyDistance), 'f', -1, 64) + "km"

	return report
}
