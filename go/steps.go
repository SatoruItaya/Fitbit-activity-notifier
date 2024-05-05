package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"time"
)

func getLifetimeStepsHistory(ctx context.Context, access_token string, today time.Time, getStepsFunc func(context.Context, string, string, string) (map[string][]map[string]string, error)) (map[time.Time]int, error) {
	// Number of target days
	restTargetDays := int(today.Sub(startDateParse).Hours() / 24)
	count := 0

	lifetimeStepsData := map[time.Time]int{}

	for restTargetDays > 0 {
		tmpEndDate := today.Add(-24 * time.Hour * time.Duration(1+LIMIT_DAYS*count))

		var tmpStartDate time.Time
		if restTargetDays > LIMIT_DAYS {
			tmpStartDate = tmpEndDate.Add(-24 * time.Hour * time.Duration(LIMIT_DAYS-1))
		} else {
			tmpStartDate = startDateParse
		}

		tmpStepsData, err := getStepsFunc(ctx, access_token, tmpStartDate.Format(DATE_FORMAT), tmpEndDate.Format(DATE_FORMAT))
		if err != nil {
			return nil, err
		}

		for _, dailyHistory := range tmpStepsData["activities-steps"] {
			dateTime, err := time.Parse(DATE_FORMAT, dailyHistory["dateTime"])
			if err != nil {
				return nil, err
			}

			dateTime = time.Date(dateTime.Year(), dateTime.Month(), dateTime.Day(), 0, 0, 0, 0, time.Local)

			step, err := strconv.Atoi(dailyHistory["value"])
			if err != nil {
				return nil, err
			}

			lifetimeStepsData[dateTime] = step
		}

		restTargetDays -= LIMIT_DAYS
		count += 1
	}

	return lifetimeStepsData, nil
}

func getStepsByDateRange(ctx context.Context, access_token string, startDate string, endDate string) (map[string][]map[string]string, error) {
	apiUrl := "https://api.fitbit.com/1/user/-/activities/steps/date/" + startDate + "/" + endDate + ".json"
	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Fitbit API request: %v", err)
	}

	req.Header.Add("Authorization", "Bearer "+access_token)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Fitbit API: %v", err)
	}
	defer resp.Body.Close()

	var responseData map[string][]map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		return nil, fmt.Errorf("failed to decode Fitbit API response: %v", err)
	}

	return responseData, nil
}

func generateStepsReport(lifetimeStepsData map[time.Time]int, today time.Time) string {
	yeatStartData := time.Date(today.Year(), time.January, 1, 0, 0, 0, 0, today.Location()).Add(-time.Nanosecond)

	// create weekly report
	targetData := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location()).AddDate(0, 0, -7)
	weeklyReport := "Weekly Report\n\n"
	weeklyTotalStep := 0
	for i := 0; i < 7; i++ {
		weeklyReport += targetData.Format(YEARLY_REPORT_DATE_FORMAT) + " " + targetData.Format(DAY_OF_WEEK_FORMAT) + " " + formatNumberWithComma(lifetimeStepsData[targetData]) + "\n"
		weeklyTotalStep += lifetimeStepsData[targetData]
		targetData = targetData.AddDate(0, 0, 1)
	}

	floatWeeklyAvetageSteps := float64(weeklyTotalStep) / float64(7)
	roundedWeeklyAvetageSteps := math.Round(floatWeeklyAvetageSteps*10) / 10
	intWeeklyAvetageSteps := int(roundedWeeklyAvetageSteps)

	weeklyReport += "\n"
	weeklyReport += "Total: " + formatNumberWithComma(weeklyTotalStep) + "\n"
	weeklyReport += "Average: " + formatNumberWithComma(intWeeklyAvetageSteps) + "\n"

	// create sorted Steps{} list by steps
	var items []Steps
	for k, v := range lifetimeStepsData {
		items = append(items, Steps{k, v})
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Value == items[j].Value {
			return items[i].Date.Before(items[j].Date)
		}
		return items[i].Value > items[j].Value
	})

	yearlyTop5Report := "Top Records in This Year\n\n"
	lifetimeTop5Report := "Top Records in Lifetime\n\n"
	yealyDataCount := 0
	count := 0

	for yealyDataCount < 5 {
		//extract yearly top5 data
		if items[count].Date.After(yeatStartData) {
			yearlyTop5Report += formatNumberWithComma(items[count].Value) + "(" + items[count].Date.Format(YEARLY_REPORT_DATE_FORMAT) + ")\n"
			yealyDataCount += 1
		}

		//extract lifetime top5 data
		if count < 5 {
			lifetimeTop5Report += formatNumberWithComma(items[count].Value) + "(" + items[count].Date.Format(LIFETIME_REPORT_DATE_FORMAT) + ")\n"
		}
		count += 1
	}

	return "\n" + SEPARATOR + weeklyReport + SEPARATOR + yearlyTop5Report + SEPARATOR + lifetimeTop5Report
}
