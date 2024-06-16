package main

import (
	"context"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

const (
	LIMIT_DAYS                  int = 1095
	DATE_FORMAT                     = "2006-01-02"
	YEARLY_REPORT_DATE_FORMAT       = "1/2"
	LIFETIME_REPORT_DATE_FORMAT     = "2006/1/2"
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

	lifetimeStepsData, err := getLifetimeStepsHistory(context.TODO(), *newAccessToken, today, getStepsByDateRange)
	if err != nil {
		return err
	}

	// if data is missing
	if len(lifetimeStepsData) <= 7 {
		return nil
	}

	stepsReport := generateStepsReport(lifetimeStepsData, today)

	lineNotifyTokenParameterName := os.Getenv("LINE_NOTIFY_TOKEN_PARAMETER_NAME")
	lineNotifyToken, err := instances.getParameter(lineNotifyTokenParameterName)
	if err != nil {
		return err
	}

	err = sendReport(*lineNotifyToken, stepsReport)
	if err != nil {
		return err
	}

	activityList, err := getActivityList(context.TODO(), *newAccessToken, today)
	if err != nil {
		return err
	}

	yearlyRunningLog, err := extractRunningLog(activityList, today)
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
