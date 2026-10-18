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
