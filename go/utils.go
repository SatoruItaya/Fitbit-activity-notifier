package main

import (
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

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
