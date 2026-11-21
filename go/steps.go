package main

import (
	"context"
	"math"
	"net/http"
	"sort"
	"strconv"
	"time"
)

func getLifetimeStepsHistory(ctx context.Context, access_token string, today time.Time, getStepsFunc func(context.Context, string, time.Time, time.Time) (map[time.Time]int, error)) (map[time.Time]int, error) {
	// Target range is [startDate, today)
	startDate := time.Date(startDateParse.Year(), startDateParse.Month(), startDateParse.Day(), 0, 0, 0, 0, time.Local)
	endDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.Local)

	lifetimeStepsData := map[time.Time]int{}

	// Google Health API accepts up to LIMIT_DAYS days per request
	for tmpEndDate := endDate; tmpEndDate.After(startDate); {
		tmpStartDate := tmpEndDate.AddDate(0, 0, -LIMIT_DAYS)
		if tmpStartDate.Before(startDate) {
			tmpStartDate = startDate
		}

		tmpStepsData, err := getStepsFunc(ctx, access_token, tmpStartDate, tmpEndDate)
		if err != nil {
			return nil, err
		}

		for date, step := range tmpStepsData {
			lifetimeStepsData[date] = step
		}

		tmpEndDate = tmpStartDate
	}

	return lifetimeStepsData, nil
}

type dailyRollUpRequest struct {
	Range struct {
		Start CivilDateTime `json:"start"`
		End   CivilDateTime `json:"end"`
	} `json:"range"`
	WindowSizeDays int    `json:"windowSizeDays"`
	PageSize       int    `json:"pageSize"`
	PageToken      string `json:"pageToken,omitempty"`
}

type dailyRollUpResponse struct {
	RollupDataPoints []struct {
		CivilStartTime CivilDateTime `json:"civilStartTime"`
		// Steps is omitted for days without data
		Steps *struct {
			CountSum string `json:"countSum"`
		} `json:"steps"`
	} `json:"rollupDataPoints"`
	NextPageToken string `json:"nextPageToken"`
}

func toCivilDateTime(t time.Time) CivilDateTime {
	return CivilDateTime{Date: CivilDate{Year: t.Year(), Month: int(t.Month()), Day: t.Day()}}
}

// getStepsByDateRange returns daily steps in [startDate, endDate).
func getStepsByDateRange(ctx context.Context, access_token string, startDate time.Time, endDate time.Time) (map[time.Time]int, error) {
	apiUrl := healthAPIBaseURL + "/steps/dataPoints:dailyRollUp"

	requestBody := dailyRollUpRequest{
		WindowSizeDays: 1,
		PageSize:       LIMIT_DAYS,
	}
	requestBody.Range.Start = toCivilDateTime(startDate)
	requestBody.Range.End = toCivilDateTime(endDate)

	stepsData := map[time.Time]int{}

	for {
		var responseData dailyRollUpResponse
		if err := callHealthAPI(ctx, access_token, http.MethodPost, apiUrl, requestBody, &responseData); err != nil {
			return nil, err
		}

		for _, dataPoint := range responseData.RollupDataPoints {
			d := dataPoint.CivilStartTime.Date
			dateTime := time.Date(d.Year, time.Month(d.Month), d.Day, 0, 0, 0, 0, time.Local)

			step := 0
			if dataPoint.Steps != nil {
				var err error
				step, err = strconv.Atoi(dataPoint.Steps.CountSum)
				if err != nil {
					return nil, err
				}
			}

			stepsData[dateTime] = step
		}

		if responseData.NextPageToken == "" {
			break
		}
		requestBody.PageToken = responseData.NextPageToken
	}

	return stepsData, nil
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
