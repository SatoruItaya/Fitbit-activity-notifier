package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Base URL of Google Health API data types. It is a variable to be replaced in tests.
var healthAPIBaseURL = "https://health.googleapis.com/v4/users/me/dataTypes"

type CivilDate struct {
	Year  int `json:"year"`
	Month int `json:"month"`
	Day   int `json:"day"`
}

type CivilTime struct {
	Hours   int `json:"hours"`
	Minutes int `json:"minutes"`
	Seconds int `json:"seconds"`
	Nanos   int `json:"nanos"`
}

type CivilDateTime struct {
	Date CivilDate `json:"date"`
	Time CivilTime `json:"time"`
}

func callHealthAPI(ctx context.Context, accessToken string, method string, apiUrl string, requestBody interface{}, responseBody interface{}) error {
	var body io.Reader
	if requestBody != nil {
		b, err := json.Marshal(requestBody)
		if err != nil {
			return fmt.Errorf("failed to encode Google Health API request: %v", err)
		}
		body = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, apiUrl, body)
	if err != nil {
		return fmt.Errorf("failed to create Google Health API request: %v", err)
	}
	req.Header.Add("Authorization", "Bearer "+accessToken)
	req.Header.Add("Accept", "application/json")
	if requestBody != nil {
		req.Header.Add("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call Google Health API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("google Health API returned status %d: %s", resp.StatusCode, string(b))
	}

	if err := json.NewDecoder(resp.Body).Decode(responseBody); err != nil {
		return fmt.Errorf("failed to decode Google Health API response: %v", err)
	}

	return nil
}
