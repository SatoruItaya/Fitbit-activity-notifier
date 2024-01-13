package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
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
	LIMIT_DAYS         int = 1095
	DATE_FORMAT            = "2006-01-02"
	REPORT_DATE_FORMAT     = "1/2"
	DAY_OF_WEEK_FORMAT     = "Mon"
	SEPARATOR              = "======================"
	DECIMAL_PLACES         = 2
)

var (
	startDate           = os.Getenv("START_DATE")
	startDateParse, _   = time.Parse(DATE_FORMAT, startDate)
	refreshCbBucketName = aws.String(os.Getenv("REFRESH_CB_BUCKET_NAME"))
	refreshCbFileName   = aws.String(os.Getenv("REFRESH_CB_FILE_NAME_GO"))
)

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

	//lifetimeStepsDataMap, err := getLifetimeStepsHistory(context.TODO(), *newAccessToken)
	_, err = getLifetimeStepsHistory(context.TODO(), *newAccessToken, today)
	if err != nil {
		return err
	}

	yearlyRunningLog, err := getYearlyRunningLog(context.TODO(), *newAccessToken, today)
	if err != nil {
		return err
	}

	err = sendRunningReport(yearlyRunningLog, today)
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

func getLifetimeStepsHistory(ctx context.Context, access_token string, today time.Time) (map[string]string, error) {
	// Number of target days
	restTargetDays := int(today.Sub(startDateParse).Hours() / 24)
	count := 0

	lifetimeStepsDataMap := map[string]string{}

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
			lifetimeStepsDataMap[dailyHistory["dateTime"]] = dailyHistory["value"]
		}

		restTargetDays -= LIMIT_DAYS
		count += 1
	}

	return lifetimeStepsDataMap, nil
}

func getStepsByDateRange(ctx context.Context, access_token string, startDate string, endDate string) (map[string][]map[string]string, error) {
	api_url := "https://api.fitbit.com/1/user/-/activities/steps/date/" + startDate + "/" + endDate + ".json"
	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, api_url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Fitbit API request: %v", err)
	}

	req.Header.Add("Authorization", "Bearer "+access_token)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Fitbit API: %v", err)
	}
	defer resp.Body.Close()

	var response_data map[string][]map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&response_data); err != nil {
		return nil, fmt.Errorf("failed to decode Fitbit API response: %v", err)
	}

	return response_data, nil
}

func getYearlyRunningLog(ctx context.Context, access_token string, today time.Time) (map[time.Time]float64, error) {
	api_url := "https://api.fitbit.com/1/user/-/activities/list.json"
	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, api_url, nil)
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

func sendRunningReport(yearlyRunningLog map[time.Time]float64, today time.Time) error {
	var keys []time.Time
	for key := range yearlyRunningLog {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].Before(keys[j])
	})

	yearlyDistance := 0.0
	weeklyDistance := 0.0
	report := SEPARATOR + "\nRunning Report\n"
	startDate := today.AddDate(0, 0, -7).Add(-time.Nanosecond)
	endDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	for _, k := range keys {
		if k.After(startDate) && k.Before(endDate) {
			report += k.Format(REPORT_DATE_FORMAT) + " " + k.Format(DAY_OF_WEEK_FORMAT) + " " + strconv.FormatFloat(roundToDecimal(yearlyRunningLog[k]), 'f', -1, 64) + "km\n"
			weeklyDistance += yearlyRunningLog[k]
		}
		yearlyDistance += yearlyRunningLog[k]
	}

	report += "\n"
	report += "Weekly Distance: " + strconv.FormatFloat(roundToDecimal(weeklyDistance), 'f', -1, 64) + "km\n"
	report += "Yearly Distance: " + strconv.FormatFloat(roundToDecimal(yearlyDistance), 'f', -1, 64) + "km\n"

	return nil
}

func roundToDecimal(num float64) float64 {
	shift := math.Pow(10, float64(DECIMAL_PLACES))
	return math.Round(num*shift) / shift
}
