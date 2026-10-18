package omwei

import (
	"fmt"
	"math"
	"math/big"
	"strings"
)

// Deterministic decimal-string conversion for the encoded integer domain.
//
// No binary floating-point arithmetic is used. Input is plain base-10 notation
// (optional sign, digits, optional '.' followed by at least one digit; no
// exponent). Exact halfway cases round away from zero. Intermediate arithmetic
// is exact and bounded to the signed 128-bit range, matching the Rust crate.

// ConversionResult is a decimal conversion result.
type ConversionResult struct {
	Atom Atom
	// Rounded is true when the scaled value was not an integer, so rounding
	// discarded non-zero fractional information (even if the integer is
	// unchanged). False means the input was exactly representable.
	Rounded bool
}

// FromDecimal converts a plain decimal string in the descriptor's canonical
// physical unit into a validated Atom. It discards the rounding report; use
// FromDecimalWithReport when precision information matters.
func FromDecimal(descriptorID uint16, input string) (Atom, error) {
	r, err := FromDecimalWithReport(descriptorID, input)
	return r.Atom, err
}

// FromDecimalWithReport is FromDecimal plus a report of whether rounding was
// required. Out-of-range results, unknown descriptors, malformed decimals and
// arithmetic overflow are rejected.
func FromDecimalWithReport(descriptorID uint16, input string) (ConversionResult, error) {
	d, ok := Lookup(descriptorID)
	if !ok {
		return ConversionResult{}, ErrUnknownDescriptor
	}
	encoded, rounded, err := convertWithParameters(input, d.Scale, d.Offset, d.Min, d.Max)
	if err != nil {
		return ConversionResult{}, err
	}
	atom, err := NewAtom(descriptorID, encoded)
	if err != nil {
		return ConversionResult{}, err
	}
	return ConversionResult{Atom: atom, Rounded: rounded}, nil
}

var (
	i128Max = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 127), big.NewInt(1))
	i128Min = new(big.Int).Neg(new(big.Int).Lsh(big.NewInt(1), 127))
)

func fitsI128(v *big.Int) bool { return v.Cmp(i128Min) >= 0 && v.Cmp(i128Max) <= 0 }

func invalid(msg string) error { return fmt.Errorf("%w: %s", ErrInvalidDecimal, msg) }

func isASCIIDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// parseDecimal returns numerator/denominator where denominator is 10^fraction digits.
func parseDecimal(input string) (num, den *big.Int, err error) {
	if input == "" {
		return nil, nil, invalid("empty decimal value")
	}
	negative := false
	unsigned := input
	switch input[0] {
	case '-':
		negative, unsigned = true, input[1:]
	case '+':
		unsigned = input[1:]
	}
	if unsigned == "" {
		return nil, nil, invalid("decimal value has no digits")
	}
	whole, fraction, hasDot := strings.Cut(unsigned, ".")
	if strings.Contains(fraction, ".") || whole == "" || !isASCIIDigits(whole) || !isASCIIDigits(fraction) {
		return nil, nil, invalid("decimal must use plain base-10 notation")
	}
	if hasDot && fraction == "" {
		return nil, nil, invalid("decimal point must be followed by digits")
	}
	if len(fraction) > 18 {
		return nil, nil, invalid("decimal precision exceeds 18 fractional digits")
	}
	// Bound the digit count before big-int parsing (i128 holds at most 39 digits).
	if len(strings.TrimLeft(whole, "0")) > 39 {
		return nil, nil, invalid("decimal value is too large")
	}
	den = new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(len(fraction))), nil)
	num, _ = new(big.Int).SetString(whole+fraction, 10)
	if negative {
		num.Neg(num)
	}
	if !fitsI128(num) {
		return nil, nil, invalid("decimal value is too large")
	}
	return num, den, nil
}

// convertWithParameters converts decimal input with an explicit integer
// multiplier, physical offset and encoded-domain bounds. The bool reports
// whether the scaled value had a non-zero fractional remainder before rounding.
func convertWithParameters(input string, scale, physicalOffset, minEncoded, maxEncoded int32) (int32, bool, error) {
	if scale <= 0 {
		return 0, false, fmt.Errorf("descriptor scale multiplier must be positive")
	}
	if minEncoded > maxEncoded {
		return 0, false, fmt.Errorf("invalid descriptor range")
	}
	num, den, err := parseDecimal(input)
	if err != nil {
		return 0, false, err
	}
	overflow := fmt.Errorf("%w: decimal arithmetic overflow", ErrOutOfRange)

	offsetNum := new(big.Int).Mul(big.NewInt(int64(physicalOffset)), den)
	shifted := new(big.Int).Sub(num, offsetNum)
	if !fitsI128(offsetNum) || !fitsI128(shifted) {
		return 0, false, overflow
	}
	scaled := new(big.Int).Mul(shifted, big.NewInt(int64(scale)))
	if !fitsI128(scaled) {
		return 0, false, overflow
	}

	magnitude := new(big.Int).Abs(scaled)
	rounded, rem := new(big.Int).QuoRem(magnitude, den, new(big.Int))
	if new(big.Int).Lsh(rem, 1).Cmp(den) >= 0 { // 2*rem >= den: ties away from zero
		rounded.Add(rounded, big.NewInt(1))
	}
	if scaled.Sign() < 0 {
		rounded.Neg(rounded)
	}
	if !rounded.IsInt64() || rounded.Int64() < math.MinInt32 || rounded.Int64() > math.MaxInt32 {
		return 0, false, fmt.Errorf("%w: encoded value overflows int32", ErrOutOfRange)
	}
	encoded := int32(rounded.Int64())
	if encoded < minEncoded || encoded > maxEncoded {
		return 0, false, fmt.Errorf("%w: encoded value is outside descriptor range", ErrOutOfRange)
	}
	return encoded, rem.Sign() != 0, nil
}
