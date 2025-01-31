package main

import (
	"math"
	"strconv"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
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

func sendReports(token string, userID string, reports []string) error {
	bot, err := messaging_api.NewMessagingApiAPI(
		token,
	)
	if err != nil {
		return err
	}

	var messageInterfaces []messaging_api.MessageInterface
	for _, msg := range reports {
		messageInterfaces = append(messageInterfaces, messaging_api.TextMessage{
			Text: msg,
		})
	}

	_, err = bot.PushMessage(
		&messaging_api.PushMessageRequest{
			To:       userID,
			Messages: messageInterfaces,
		},
		"",
	)
	if err != nil {
		return err
	}

	return nil
}
