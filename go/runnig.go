package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"
)

func getYearlyRunningLog(ctx context.Context, access_token string, today time.Time) (map[time.Time]float64, error) {
	apiUrl := "https://api.fitbit.com/1/user/-/activities/list.json"
	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Fitbit API request: %v", err)
	}
	req.Header.Add("Authorization", "Bearer "+access_token)

	thisYear, _ := strconv.Atoi(today.Format("2006"))
	targetDate := today
	baseDate := time.Date(thisYear-1, time.December, 31, 23, 59, 59, 999, time.UTC)

	yearlyRunningLog := map[time.Time]float64{}

	for targetDate.After(baseDate) {
		query := req.URL.Query()
		query.Set("beforeDate", targetDate.Format(DATE_FORMAT))
		query.Set("sort", "desc")
		query.Set("limit", "100")
		query.Set("offset", "0")
		req.URL.RawQuery = query.Encode()

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to call Fitbit API: %v", err)
		}
		defer resp.Body.Close()

		var activityLogList map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&activityLogList); err != nil {
			return nil, fmt.Errorf("failed to decode Fitbit API response: %v", err)
		}

		activities, ok := activityLogList["activities"].([]interface{})
		if !ok {
			return nil, errors.New("unable to extract activities")
		}

		for i, a := range activities {
			activity := a.(map[string]interface{})
			activityName, ok := activity["activityName"].(string)
			if !ok {
				return nil, errors.New("unable to extract activityName")
			}

			t, ok := activity["startTime"].(string)
			if !ok {
				return nil, errors.New("unable to extract date")
			}
			startTime, err := time.Parse("2006-01-02T15:04:05.000-07:00", t)
			if err != nil {
				return nil, err
			}

			if activityName == "Run" {
				distance, ok := activity["distance"].(float64)
				if !ok {
					return nil, errors.New("unable to extract distance")
				}

				if startTime.After(baseDate) {
					yearlyRunningLog[startTime] = distance
				}
			}

			if i == len(activities)-1 {
				targetDate = startTime
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
	weekStartDate := today.AddDate(0, 0, -7).Add(-time.Nanosecond)
	weekEndDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	report := "\n" + SEPARATOR + "Running Report\n"

	for _, k := range keys {
		if k.After(weekStartDate) && k.Before(weekEndDate) {
			report += k.Format(YEARLY_REPORT_DATE_FORMAT) + " " + k.Format(DAY_OF_WEEK_FORMAT) + " " + strconv.FormatFloat(roundToDecimal(yearlyRunningLog[k]), 'f', -1, 64) + "km\n"
			weeklyDistance += yearlyRunningLog[k]
		}
		yearlyDistance += yearlyRunningLog[k]
	}

	report += "\n"
	report += "Weekly Distance: " + strconv.FormatFloat(roundToDecimal(weeklyDistance), 'f', -1, 64) + "km\n"
	report += "Yearly Distance: " + strconv.FormatFloat(roundToDecimal(yearlyDistance), 'f', -1, 64) + "km\n"

	return report
}
