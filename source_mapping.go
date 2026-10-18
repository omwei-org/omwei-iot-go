package omwei

import (
	"fmt"
	"math"
	"math/big"
)

// SourceIdentity identifies one concrete point in an upstream system.
// Point is a protocol-specific address or node identifier.
type SourceIdentity struct {
	Protocol string `json:"protocol"`
	Device   string `json:"device"`
	Point    string `json:"point"`
}

// SourceMapping transforms a raw source value into the canonical physical unit
// of DescriptorID. The transform is canonical = raw*Scale + Offset, with both
// scale and offset represented as exact rational numbers. SourceUnit describes
// the raw value; CanonicalUnit must match the descriptor's canonical unit.
// Unit conversion is not inferred: the configured transform must perform it.
type SourceMapping struct {
	Source            SourceIdentity `json:"source"`
	DescriptorID      uint16         `json:"descriptor_id"`
	SourceUnit        string         `json:"source_unit"`
	CanonicalUnit     string         `json:"canonical_unit"`
	ScaleNumerator    int64          `json:"scale_numerator"`
	ScaleDenominator  int64          `json:"scale_denominator"`
	OffsetNumerator   int64          `json:"offset_numerator"`
	OffsetDenominator int64          `json:"offset_denominator"`
}

// MapSourceValue applies an explicit source transform and constructs a validated
// atom. rawValue is a plain decimal string to avoid binary floating-point drift.
func MapSourceValue(mapping SourceMapping, rawValue string) (ConversionResult, error) {
	if err := ValidateSourceMapping(mapping); err != nil {
		return ConversionResult{}, err
	}
	d, _ := Lookup(mapping.DescriptorID)
	rawNum, rawDen, err := parseDecimal(rawValue)
	if err != nil {
		return ConversionResult{}, err
	}

	scaleNum := big.NewInt(mapping.ScaleNumerator)
	scaleDen := big.NewInt(mapping.ScaleDenominator)
	offsetNum := big.NewInt(mapping.OffsetNumerator)
	offsetDen := big.NewInt(mapping.OffsetDenominator)

	// Combine raw*scale + offset as one exact rational.
	physicalNum := new(big.Int).Mul(rawNum, scaleNum)
	physicalNum.Mul(physicalNum, offsetDen)
	offsetTerm := new(big.Int).Mul(offsetNum, rawDen)
	offsetTerm.Mul(offsetTerm, scaleDen)
	physicalNum.Add(physicalNum, offsetTerm)
	physicalDen := new(big.Int).Mul(rawDen, scaleDen)
	physicalDen.Mul(physicalDen, offsetDen)

	// Descriptor encoding: round((physical - descriptor offset) * scale).
	shift := new(big.Int).Mul(big.NewInt(int64(d.Offset)), physicalDen)
	shift.Sub(physicalNum, shift)
	encodedNum := new(big.Int).Mul(shift, big.NewInt(int64(d.Scale)))
	encoded, rounded := roundRationalAwayFromZero(encodedNum, physicalDen)
	if !encoded.IsInt64() || encoded.Int64() < math.MinInt32 || encoded.Int64() > math.MaxInt32 {
		return ConversionResult{}, fmt.Errorf("%w: encoded value overflows int32", ErrOutOfRange)
	}
	atom, err := NewAtom(d.ID, int32(encoded.Int64()))
	if err != nil {
		return ConversionResult{}, err
	}
	return ConversionResult{Atom: atom, Rounded: rounded}, nil
}

// roundRationalAwayFromZero rounds to the nearest integer; halfway cases are
// rounded away from zero. denominator must be positive.
func roundRationalAwayFromZero(numerator, denominator *big.Int) (*big.Int, bool) {
	magnitude := new(big.Int).Abs(numerator)
	quotient, remainder := new(big.Int).QuoRem(magnitude, denominator, new(big.Int))
	rounded := remainder.Sign() != 0
	if new(big.Int).Lsh(new(big.Int).Set(remainder), 1).Cmp(denominator) >= 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	if numerator.Sign() < 0 {
		quotient.Neg(quotient)
	}
	return quotient, rounded
}
