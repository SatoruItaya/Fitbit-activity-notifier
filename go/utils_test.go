package main

import "testing"

func TestRoundToDecimal(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{3.14159, 3.14},
		{10.6789, 10.68},
	}

	for _, test := range tests {
		result := roundToDecimal(test.input)
		if result != test.expected {
			t.Errorf("For input %f, expected %f, but got %f", test.input, test.expected, result)
		}
	}
}

func TestFormatNumberWithComma(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{123, "123"},
		{1234, "1,234"},
		{123456, "123,456"},
		{1234567, "1,234,567"},
		{1234567890, "1,234,567,890"},
	}

	for _, test := range tests {
		result := formatNumberWithComma(test.input)
		if result != test.expected {
			t.Errorf("For input %d, expected %s, but got %s", test.input, test.expected, result)
		}
	}
}
