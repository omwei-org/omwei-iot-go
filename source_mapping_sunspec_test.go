package omwei

import (
	"errors"
	"testing"
)

func TestDecodeSunSpecScaledInteger(t *testing.T) {
	tests := []struct {
		name string
		raw int64
		sf int16
		want string
	}{
		{"one decimal place", 1234, -1, "123.4"},
		{"two decimal places", 1234, -2, "12.34"},
		{"pad fractional zeroes", 5, -2, "0.05"},
		{"positive exponent", 123, 2, "12300"},
		{"negative value", -1234, -1, "-123.4"},
		{"zero", 0, -3, "0"},
		{"no scale", 123, 0, "123"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeSunSpecScaledInteger(tt.raw, tt.sf)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("want %q, got %q", tt.want, got)
			}
		})
	}
}

func TestDecodeSunSpecScaledIntegerRejectsExtremeScaleFactor(t *testing.T) {
	for _, sf := range []int16{-19, 19} {
		if _, err := DecodeSunSpecScaledInteger(1, sf); !errors.Is(err, ErrInvalidSunSpecScaleFactor) {
			t.Fatalf("sf=%d: want invalid scale factor, got %v", sf, err)
		}
	}
}
