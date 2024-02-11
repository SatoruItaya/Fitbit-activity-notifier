package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

type Instances struct {
	SSMClient *ssm.Client
	S3Client  *s3.Client
}

const (
	LIMIT_DAYS                  int = 1095
	DATE_FORMAT                     = "2006-01-02"
	YEARLY_REPORT_DATE_FORMAT       = "1/2"
	LIFETIME_REPORT_DATE_FORMAT     = "2006/01/02"
	DAY_OF_WEEK_FORMAT              = "Mon"
	SEPARATOR                       = "======================\n"
	DECIMAL_PLACES                  = 2
)

var (
	startDate           = os.Getenv("START_DATE")
	startDateParse, _   = time.Parse(DATE_FORMAT, startDate)
	refreshCbBucketName = aws.String(os.Getenv("REFRESH_CB_BUCKET_NAME"))
	refreshCbFileName   = aws.String(os.Getenv("REFRESH_CB_FILE_NAME_GO"))
)

type Steps struct {
	Date  time.Time
	Value int
}

func main() {
	lambda.Start(handler)
}

func handler() error {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return err
	}

	instances := &Instances{
		SSMClient: ssm.NewFromConfig(cfg),
		S3Client:  s3.NewFromConfig(cfg),
	}

	clientIDParameterName := os.Getenv("CLIENT_ID_PARAMETER_NAME_GO")
	clientID, err := instances.getParameter(clientIDParameterName)
	if err != nil {
		return err
	}

	clientSecretParameterName := os.Getenv("CLIENT_SECRET_PARAMETER_NAME_GO")
	clientSecret, err := instances.getParameter(clientSecretParameterName)
	if err != nil {
		return err
	}

	refreshToken, err := instances.getRefreshToken()
	if err != nil {
		return err
	}

	newAccessToken, err := instances.refreshAccessToken(context.TODO(), *clientID, *clientSecret, *refreshToken)
	if err != nil {
		return err
	}

	today := time.Now().Local()

	lifetimeStepsData, err := getLifetimeStepsHistory(context.TODO(), *newAccessToken, today)
	if err != nil {
		return err
	}

	lineNotifyTokenParameterName := os.Getenv("LINE_NOTIFY_TOKEN_PARAMETER_NAME")
	lineNotifyToken, err := instances.getParameter(lineNotifyTokenParameterName)
	if err != nil {
		return err
	}

	stepsReport := generateStepsReport(lifetimeStepsData, today)

	err = sendReport(*lineNotifyToken, stepsReport)
	if err != nil {
		return err
	}

	yearlyRunningLog, err := getYearlyRunningLog(context.TODO(), *newAccessToken, today)
	if err != nil {
		return err
	}

	runningReport := generateRunningReport(yearlyRunningLog, today)

	err = sendReport(*lineNotifyToken, runningReport)
	if err != nil {
		return err
	}

	return nil
}

func (instances *Instances) getParameter(parameterName string) (*string, error) {
	getParameterOutput, err := instances.SSMClient.GetParameter(context.TODO(), &ssm.GetParameterInput{
		Name:           aws.String(parameterName),
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return nil, err
	}

	return getParameterOutput.Parameter.Value, nil
}

func (instances *Instances) getRefreshToken() (*string, error) {
	getObjectOutput, err := instances.S3Client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: refreshCbBucketName,
		Key:    refreshCbFileName,
	})
	if err != nil {
		return nil, err
	}

	defer getObjectOutput.Body.Close()
	body, err := io.ReadAll(getObjectOutput.Body)
	if err != nil {
		return nil, err
	}

	refreshToken := string(body)

	return &refreshToken, nil
}

func getFitbitConfig(clientID string, clientSecret string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint: oauth2.Endpoint{
			TokenURL: "https://api.fitbit.com/oauth2/token",
		},
	}
}

func (instances *Instances) refreshAccessToken(ctx context.Context, clientID string, clientSecret string, refreshToken string) (*string, error) {
	config := getFitbitConfig(clientID, clientSecret)
	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}

	newToken, err := config.TokenSource(ctx, token).Token()
	if err != nil {
		return nil, err
	}

	tempFileName := "/tmp/" + *refreshCbBucketName

	tempFile, err := os.Create(tempFileName)
	if err != nil {
		return nil, err
	}
	defer tempFile.Close()

	newRefreshToken := newToken.RefreshToken
	data := []byte(newRefreshToken)
	_, err = tempFile.Write(data)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(tempFileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	_, err = instances.S3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: refreshCbBucketName,
		Key:    refreshCbFileName,
		Body:   file,
	})
	if err != nil {
		return nil, err
	}

	return &newToken.AccessToken, nil
}

func getLifetimeStepsHistory(ctx context.Context, access_token string, today time.Time) (map[time.Time]int, error) {
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

		tmpStepsData, err := getStepsByDateRange(context.TODO(), access_token, tmpStartDate.Format(DATE_FORMAT), tmpEndDate.Format(DATE_FORMAT))
		if err != nil {
			return nil, err
		}

		for _, dailyHistory := range tmpStepsData["activities-steps"] {
			dateTime, err := time.Parse(DATE_FORMAT, dailyHistory["dateTime"])
			if err != nil {
				return nil, err
			}

			//timezone指定
			dateTime = dateTime.Local()

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
		targetData = targetData.AddDate(0, 0, 1)
		weeklyTotalStep += lifetimeStepsData[targetData]
	}

	weeklyReport += "\n"
	weeklyReport += "Total: " + formatNumberWithComma(weeklyTotalStep) + "\n"
	weeklyReport += "Average: " + formatNumberWithComma(weeklyTotalStep/7) + "\n"

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
	lifetimeTop5Report := "Top Records in This Lifetime\n\n"
	yealyDataCount := 0
	count := 0

	for yealyDataCount <= 5 {
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

func roundToDecimal(num float64) float64 {
	shift := math.Pow(10, float64(DECIMAL_PLACES))
	return math.Round(num*shift) / shift
}

func formatNumberWithComma(number int) string {
	str := strconv.FormatInt(int64(number), 10)
	result := ""
	for i := len(str); i > 0; i -= 3 {
		if i-3 > 0 {
			result = "," + str[i-3:i] + result
		} else {
			result = str[:i] + result
		}
	}
	return result
}

func sendReport(token string, report string) error {
	apiUrl := "https://notify-api.line.me/api/notify"
	u, err := url.ParseRequestURI(apiUrl)
	if err != nil {
		log.Fatal(err)
	}

	c := &http.Client{}
	form := url.Values{}
	form.Add("message", report)
	body := strings.NewReader(form.Encode())
	req, err := http.NewRequest("POST", u.String(), body)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+token)
	_, err = c.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	return nil
}
