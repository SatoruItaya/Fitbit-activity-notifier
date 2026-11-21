package main

import (
	"context"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"golang.org/x/oauth2"
)

type Instances struct {
	SSMClient *ssm.Client
	S3Client  *s3.Client
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

	refreshToken := strings.TrimSpace(string(body))

	return &refreshToken, nil
}

func getGoogleOAuthConfig(clientID string, clientSecret string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint: oauth2.Endpoint{
			TokenURL:  "https://oauth2.googleapis.com/token",
			AuthStyle: oauth2.AuthStyleInParams,
		},
	}
}

func (instances *Instances) refreshAccessToken(ctx context.Context, clientID string, clientSecret string, refreshToken string) (*string, error) {
	config := getGoogleOAuthConfig(clientID, clientSecret)
	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}

	newToken, err := config.TokenSource(ctx, token).Token()
	if err != nil {
		return nil, err
	}

	// Google usually does not rotate refresh tokens, so save it only when a new one is issued
	if newToken.RefreshToken != "" && newToken.RefreshToken != refreshToken {
		_, err = instances.S3Client.PutObject(ctx, &s3.PutObjectInput{
			Bucket: refreshCbBucketName,
			Key:    refreshCbFileName,
			Body:   strings.NewReader(newToken.RefreshToken),
		})
		if err != nil {
			return nil, err
		}
	}

	return &newToken.AccessToken, nil
}
