package omwei

import (
	"errors"
	"math/big"
	"strings"
)

var ErrInvalidSunSpecScaleFactor = errors.New("invalid SunSpec scale factor")

// DecodeSunSpecScaledInteger applies the SunSpec scale-factor rule
// engineering_value = raw * 10^scaleFactor without floating-point arithmetic.
// The returned decimal string can be passed directly to MapSourceValue.
// The supported exponent range [-18, 18] is a defensive implementation bound.
func DecodeSunSpecScaledInteger(raw int64, scaleFactor int16) (string, error) {
	if scaleFactor < -18 || scaleFactor > 18 {
		return "", ErrInvalidSunSpecScaleFactor
	}
	value := big.NewInt(raw)
	negative := value.Sign() < 0
	value.Abs(value)
	digits := value.String()

	if scaleFactor >= 0 {
		multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scaleFactor)), nil)
		value.Mul(value, multiplier)
		digits = value.String()
	} else {
		places := int(-scaleFactor)
		if len(digits) <= places {
			digits = strings.Repeat("0", places-len(digits)+1) + digits
		}
		cut := len(digits) - places
		digits = digits[:cut] + "." + digits[cut:]
		digits = strings.TrimRight(digits, "0")
		digits = strings.TrimRight(digits, ".")
		if digits == "" {
			digits = "0"
		}
	}
	if negative && digits != "0" {
		digits = "-" + digits
	}
	return digits, nil
}

// MapSunSpecModel103ActivePower applies W_SF before canonical mapping.
func MapSunSpecModel103ActivePower(mapping SourceMapping, source SourceIdentity, raw int64, sf int16) (ConversionResult, error) {
	if source != mapping.Source {
		return ConversionResult{}, ErrSourceIdentityMismatch
	}
	if source.Protocol != "sunspec-modbus" || source.Point != "model:103:W;scale-factor:W_SF" || mapping.DescriptorID != 69 {
		return ConversionResult{}, ErrInvalidSourceMapping
	}
	if err := ValidateSourceMapping(mapping); err != nil {
		return ConversionResult{}, err
	}
	if mapping.ScaleNumerator != 1 || mapping.ScaleDenominator != 1 || mapping.OffsetNumerator != 0 || mapping.OffsetDenominator != 1 {
		return ConversionResult{}, ErrInvalidSourceMapping
	}
	value, err := DecodeSunSpecScaledInteger(raw, sf)
	if err != nil {
		return ConversionResult{}, err
	}
	return MapSourceSample(mapping, SourceSample{Source: source, Value: value, Unit: mapping.SourceUnit})
}
 
// SunSpecModel103ActivePowerSnapshot is one coherent adapter read. The adapter
// must populate W and W_SF from the same model instance/read transaction and
// use SnapshotID to identify that acquisition. This package cannot verify the
// transport's claim; it makes the claim explicit at the mapper boundary.
type SunSpecModel103ActivePowerSnapshot struct {
	Source     SourceIdentity
	SnapshotID string
	ModelID    uint16
	RawW       int16
	WSF        int16
}

// MapSunSpecModel103Snapshot maps an explicitly coherent Model 103 sample.
// It rejects absent snapshot identity, wrong model/point identity, and a
// source-device mismatch before producing an atom.
func MapSunSpecModel103Snapshot(mapping SourceMapping, sample SunSpecModel103ActivePowerSnapshot) (ConversionResult, error) {
	if strings.TrimSpace(sample.SnapshotID) == "" || sample.ModelID != 103 ||
		sample.Source.Device == "" || sample.Source != mapping.Source {
		return ConversionResult{}, ErrSourceIdentityMismatch
	}
	if sample.Source.Protocol != "sunspec-modbus" ||
		sample.Source.Point != "model:103:W;scale-factor:W_SF" {
		return ConversionResult{}, ErrInvalidSourceMapping
	}
	return MapSunSpecModel103ActivePower(mapping, sample.Source, int64(sample.RawW), sample.WSF)
}

