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

func TestMapSunSpecModel103ActivePowerEndToEnd(t *testing.T) {
	mapping := SourceMapping{
		Source: SourceIdentity{Protocol: "sunspec-modbus", Device: "inverter-2", Point: "model:103:W;scale-factor:W_SF"},
		DescriptorID: 69,
		SourceUnit: "watt (after W_SF application)",
		CanonicalUnit: "watt",
		ScaleNumerator: 1, ScaleDenominator: 1,
		OffsetNumerator: 0, OffsetDenominator: 1,
	}
	tests := []struct {
		name string
		raw int64
		sf int16
		want int32
		wire string
	}{
		{"scale zero", 123, 0, 123, "00450000007b"},
		{"negative scale", 1234, -1, 123, "00450000007b"},
		{"positive scale", 12, 1, 120, "004500000078"},
		{"round half away from zero", -125, -1, -13, "0045fffffff3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MapSunSpecModel103ActivePower(mapping, mapping.Source, tt.raw, tt.sf)
			if err != nil { t.Fatal(err) }
			if got.Atom.CanonicalValue != tt.want { t.Fatalf("want %d, got %d", tt.want, got.Atom.CanonicalValue) }
			wire := Encode(got.Atom)
			if hex.EncodeToString(wire[:]) != tt.wire { t.Fatalf("want wire %s, got %x", tt.wire, wire) }
		})
	}
}

func TestMapSunSpecModel103ActivePowerRejectsBadIdentityAndDoubleScale(t *testing.T) {
	mapping := SourceMapping{
		Source: SourceIdentity{Protocol: "sunspec-modbus", Device: "inverter-2", Point: "model:103:W;scale-factor:W_SF"},
		DescriptorID: 69, SourceUnit: "watt (after W_SF application)", CanonicalUnit: "watt",
		ScaleNumerator: 1, ScaleDenominator: 1, OffsetNumerator: 0, OffsetDenominator: 1,
	}
	source := mapping.Source
	source.Device = "inverter-3"
	if _, err := MapSunSpecModel103ActivePower(mapping, source, 123, 0); !errors.Is(err, ErrSourceIdentityMismatch) {
		t.Fatalf("want identity mismatch, got %v", err)
	}
	if _, err := MapSunSpecModel103ActivePower(mapping, mapping.Source, 123, 19); !errors.Is(err, ErrInvalidSunSpecScaleFactor) {
		t.Fatalf("want invalid W_SF rejection, got %v", err)
	}
	mapping.ScaleNumerator = 10
	if _, err := MapSunSpecModel103ActivePower(mapping, mapping.Source, 123, -1); !errors.Is(err, ErrInvalidSourceMapping) {
		t.Fatalf("want double-scale profile rejection, got %v", err)
	}
}
