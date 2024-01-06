package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

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

	userProfile, err := callFitbitAPI(context.TODO(), *newAccessToken)
	if err != nil {
		return err
	}
	fmt.Printf("Fitbit API Response: %+v\n", userProfile)

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
		Bucket: aws.String(os.Getenv("REFRESH_CB_BUCKET_NAME")),
		Key:    aws.String(os.Getenv("REFRESH_CB_FILE_NAME_GO")),
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

	tempFileName := "/tmp/" + os.Getenv("REFRESH_CB_FILE_NAME_GO")

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
		Bucket: aws.String(os.Getenv("REFRESH_CB_BUCKET_NAME")),
		Key:    aws.String(os.Getenv("REFRESH_CB_FILE_NAME_GO")),
		Body:   file,
	})
	if err != nil {
		return nil, err
	}

	return &newToken.AccessToken, nil
}

func callFitbitAPI(ctx context.Context, access_token string) (map[string]interface{}, error) {
	api_url := "https://api.fitbit.com/1/user/-/profile.json"
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

	var response_data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response_data); err != nil {
		return nil, fmt.Errorf("failed to decode Fitbit API response: %v", err)
	}

	return response_data, nil
}
