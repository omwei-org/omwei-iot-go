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


// TestTwoIndependentSourcesMapToSameCanonicalDescriptor is an illustrative
// Modbus/SunSpec-style PoC. These profiles demonstrate the mapping contract;
// they are not asserted to match a specific vendor device's register map.
func TestTwoIndependentSourcesMapToSameCanonicalDescriptor(t *testing.T) {
	modbus := temperatureSourceMapping()
	sunspec := SourceMapping{
		Source: SourceIdentity{
			Protocol: "sunspec",
			Device:   "inverter-2",
			Point:    "model:103:TmpAmb",
		},
		DescriptorID:      AmbientTemperatureID,
		SourceUnit:        "degree_Celsius",
		CanonicalUnit:     "degree_Celsius",
		ScaleNumerator:    1,
		ScaleDenominator:  1,
		OffsetNumerator:   0,
		OffsetDenominator: 1,
	}

	modbusResult, err := MapSourceValue(modbus, "225") // register value in tenths of °C
	if err != nil {
		t.Fatal(err)
	}
	sunspecResult, err := MapSourceValue(sunspec, "22.5") // already expressed in °C
	if err != nil {
		t.Fatal(err)
	}

	if modbus.Source == sunspec.Source {
		t.Fatal("the source identities must remain distinct")
	}
	if modbusResult.Atom != sunspecResult.Atom {
		t.Fatalf("same physical value must produce same canonical atom: Modbus=%+v SunSpec=%+v", modbusResult.Atom, sunspecResult.Atom)
	}
	modbusWire, sunspecWire := Encode(modbusResult.Atom), Encode(sunspecResult.Atom)
	if modbusWire != sunspecWire {
		t.Fatalf("same canonical atom must have identical wire bytes: Modbus=%x SunSpec=%x", modbusWire, sunspecWire)
	}
}


func TestMapSourceSampleChecksIdentityAndObservedUnit(t *testing.T) {
	m := temperatureSourceMapping()
	sample := SourceSample{
		Source: m.Source,
		Value: "225",
		Unit: m.SourceUnit,
	}
	result, err := MapSourceSample(m, sample)
	if err != nil {
		t.Fatal(err)
	}
	if result.Atom != (Atom{DescriptorID: AmbientTemperatureID, CanonicalValue: 22500}) {
		t.Fatalf("unexpected mapped atom: %+v", result.Atom)
	}

	sample.Source.Point = "holding:40002"
	if _, err := MapSourceSample(m, sample); !errors.Is(err, ErrSourceIdentityMismatch) {
		t.Fatalf("want source identity mismatch, got %v", err)
	}

	sample.Source = m.Source
	sample.Unit = "degree_Fahrenheit"
	if _, err := MapSourceSample(m, sample); !errors.Is(err, ErrObservedSourceUnitMismatch) {
		t.Fatalf("want observed source unit mismatch, got %v", err)
	}
}

func TestMapSourceSampleRejectsBlankValue(t *testing.T) {
	m := temperatureSourceMapping()
	_, err := MapSourceSample(m, SourceSample{Source: m.Source, Value: " ", Unit: m.SourceUnit})
	if !errors.Is(err, ErrInvalidDecimal) {
		t.Fatalf("want invalid decimal, got %v", err)
	}
}

func TestMapSourceValueAllowsNegativeScaleAndRejectsZeroScale(t *testing.T) {
	m := temperatureSourceMapping()
	m.ScaleNumerator, m.ScaleDenominator = -1, 1
	m.SourceUnit = "degree_Celsius"
	result, err := MapSourceValue(m, "1")
	if err != nil {
		t.Fatalf("negative scale should be accepted as an explicit transform: %v", err)
	}
	if result.Atom.CanonicalValue != -1000 {
		t.Fatalf("negative scale should map 1 C to -1 C: got %+v", result.Atom)
	}

	m.ScaleNumerator = 0
	if _, err := MapSourceValue(m, "1"); !errors.Is(err, ErrInvalidSourceMapping) {
		t.Fatalf("zero scale must be rejected, got %v", err)
	}
}


func TestMapSourceSampleRejectsEveryIdentityComponentMismatch(t *testing.T) {
	m := temperatureSourceMapping()
	base := SourceSample{Source: m.Source, Value: "225", Unit: m.SourceUnit}
	tests := []struct {
		name string
		edit func(*SourceSample)
	}{
		{"protocol", func(s *SourceSample) { s.Source.Protocol = "sunspec" }},
		{"device", func(s *SourceSample) { s.Source.Device = "inverter-2" }},
		{"point", func(s *SourceSample) { s.Source.Point = "holding:40002" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sample := base
			tt.edit(&sample)
			if _, err := MapSourceSample(m, sample); !errors.Is(err, ErrSourceIdentityMismatch) {
				t.Fatalf("want source identity mismatch, got %v", err)
			}
		})
	}
}

func TestMapSourceSampleRejectsIdentityMismatchBeforeMappingValue(t *testing.T) {
	m := temperatureSourceMapping()
	sample := SourceSample{
		Source: SourceIdentity{Protocol: "modbus", Device: "other-device", Point: m.Source.Point},
		Value: "not-a-number",
		Unit: m.SourceUnit,
	}
	if _, err := MapSourceSample(m, sample); !errors.Is(err, ErrSourceIdentityMismatch) {
		t.Fatalf("identity must be checked before value parsing, got %v", err)
	}
}
