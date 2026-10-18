package omwei

import (
	"errors"
	"testing"
)

func temperatureSourceMapping() SourceMapping {
	return SourceMapping{
		Source: SourceIdentity{Protocol: "modbus", Device: "inverter-1", Point: "holding:40001"},
		DescriptorID: AmbientTemperatureID,
		SourceUnit: "register-tenths-of-degree-C",
		CanonicalUnit: "degree_Celsius",
		ScaleNumerator: 1, ScaleDenominator: 10,
		OffsetNumerator: 0, OffsetDenominator: 1,
	}
}

func TestMapSourceValueAppliesScaleAndKeepsAtomFormat(t *testing.T) {
	result, err := MapSourceValue(temperatureSourceMapping(), "225")
	if err != nil {
		t.Fatal(err)
	}
	if result.Atom != (Atom{DescriptorID: AmbientTemperatureID, CanonicalValue: 22500}) || result.Rounded {
		t.Fatalf("unexpected mapping result: %+v", result)
	}
	wire := Encode(result.Atom)
	decoded, err := Decode(wire[:])
	if err != nil || decoded != result.Atom {
		t.Fatalf("wire round trip failed: atom=%+v decoded=%+v err=%v", result.Atom, decoded, err)
	}
}

func TestMapSourceValueAppliesOffsetExactly(t *testing.T) {
	m := temperatureSourceMapping()
	m.ScaleNumerator, m.ScaleDenominator = 1, 1
	m.OffsetNumerator, m.OffsetDenominator = 41, 2
	result, err := MapSourceValue(m, "2")
	if err != nil {
		t.Fatal(err)
	}
	if result.Atom.CanonicalValue != 22500 {
		t.Fatalf("want 22.5 C encoded as 22500, got %+v", result.Atom)
	}
}

func TestMapSourceValueRejectsCanonicalUnitMismatch(t *testing.T) {
	m := temperatureSourceMapping()
	m.CanonicalUnit = "degree_Fahrenheit"
	if _, err := MapSourceValue(m, "225"); !errors.Is(err, ErrSourceUnitMismatch) {
		t.Fatalf("want canonical unit mismatch, got %v", err)
	}
}

func TestMapSourceValueRejectsIncompleteOrInvalidProfile(t *testing.T) {
	m := temperatureSourceMapping()
	m.Source.Point = ""
	if _, err := MapSourceValue(m, "225"); !errors.Is(err, ErrInvalidSourceMapping) {
		t.Fatalf("want invalid source mapping, got %v", err)
	}
	m = temperatureSourceMapping()
	m.ScaleDenominator = 0
	if _, err := MapSourceValue(m, "225"); !errors.Is(err, ErrInvalidSourceMapping) {
		t.Fatalf("want invalid scale denominator, got %v", err)
	}
	m = temperatureSourceMapping()
	m.OffsetDenominator = 0
	if _, err := MapSourceValue(m, "225"); !errors.Is(err, ErrInvalidSourceMapping) {
		t.Fatalf("want invalid offset denominator, got %v", err)
	}
}

func TestMapSourceValueRejectsInvalidValueAndOutOfRange(t *testing.T) {
	m := temperatureSourceMapping()
	if _, err := MapSourceValue(m, "2e2"); !errors.Is(err, ErrInvalidDecimal) {
		t.Fatalf("want invalid decimal, got %v", err)
	}
	if _, err := MapSourceValue(m, "999999"); !errors.Is(err, ErrOutOfRange) {
		t.Fatalf("want out-of-range, got %v", err)
	}
}

func TestMapSourceValueRoundingReport(t *testing.T) {
	m := temperatureSourceMapping()
	m.ScaleNumerator, m.ScaleDenominator = 1, 3
	result, err := MapSourceValue(m, "1")
	if err != nil {
		t.Fatal(err)
	}
	if result.Atom.CanonicalValue != 333 || !result.Rounded {
		t.Fatalf("expected rounded encoded value 333, got %+v", result)
	}
}
