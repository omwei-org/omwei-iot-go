package omwei

import "testing"

func mustDecimal(t *testing.T, input string) int32 {
	t.Helper()
	a, err := FromDecimal(AmbientTemperatureID, input)
	if err != nil {
		t.Fatalf("%q: %v", input, err)
	}
	return a.CanonicalValue
}

func TestDecimalUsesExactBaseTen(t *testing.T) {
	for _, in := range []string{"22.500", "22.5"} {
		if got := mustDecimal(t, in); got != 22_500 {
			t.Fatalf("%q -> %d", in, got)
		}
	}
}

func TestConversionReport(t *testing.T) {
	cases := []struct {
		in      string
		want    int32
		rounded bool
	}{
		{"22.5", 22_500, false}, {"22.5004", 22_500, true}, {"22.5001", 22_500, true},
		{"22.5005", 22_501, true}, {"-22.5004", -22_500, true},
	}
	for _, c := range cases {
		r, err := FromDecimalWithReport(AmbientTemperatureID, c.in)
		if err != nil || r.Atom.CanonicalValue != c.want || r.Rounded != c.rounded {
			t.Errorf("%q: %+v %v", c.in, r, err)
		}
	}
}

func TestTiesRoundAwayFromZero(t *testing.T) {
	for in, want := range map[string]int32{"0.0005": 1, "-0.0005": -1, "22.5005": 22_501, "-22.5005": -22_501} {
		if got := mustDecimal(t, in); got != want {
			t.Errorf("%q -> %d want %d", in, got, want)
		}
	}
}

func TestRangeCheckedAfterRounding(t *testing.T) {
	if mustDecimal(t, "-80.000") != -80_000 || mustDecimal(t, "150.000") != 150_000 {
		t.Fatal("bounds must be accepted")
	}
	for _, in := range []string{"-80.0005", "150.0005"} {
		if _, err := FromDecimal(AmbientTemperatureID, in); err == nil {
			t.Errorf("%q must be rejected", in)
		}
	}
}

func TestNonzeroOffsetAppliedBeforeScaling(t *testing.T) {
	// physical = encoded*0.1 + 10  =>  encoded = round((physical-10)*10), range [-100,100].
	ok := []struct {
		in      string
		want    int32
		rounded bool
	}{{"10", 0, false}, {"10.05", 1, true}, {"0", -100, false}, {"20", 100, false}}
	for _, c := range ok {
		got, rounded, err := convertWithParameters(c.in, 10, 10, -100, 100)
		if err != nil || got != c.want || rounded != c.rounded {
			t.Errorf("%q: %d %v %v", c.in, got, rounded, err)
		}
	}
	for _, in := range []string{"-0.05", "20.05"} {
		if _, _, err := convertWithParameters(in, 10, 10, -100, 100); err == nil {
			t.Errorf("%q must be rejected", in)
		}
	}
}

func TestMalformedAndExcessPrecisionFailClosed(t *testing.T) {
	for _, v := range []string{"", "+", "-", ".5", "1.", "1e2", "1.2.3", "1.0000000000000000001", " 1", "1 ", "--1", "0x10", "١٢"} {
		if _, err := FromDecimal(AmbientTemperatureID, v); err == nil {
			t.Errorf("%q must be rejected", v)
		}
	}
	if _, err := FromDecimal(0xFFFF, "1"); err == nil {
		t.Error("unknown descriptor must be rejected")
	}
}

func TestHugeInputsDoNotPanicOrWrap(t *testing.T) {
	huge := "1" + string(make([]byte, 0))
	for i := 0; i < 200; i++ {
		huge += "0"
	}
	for _, in := range []string{huge, "-" + huge, huge + ".5"} {
		if _, err := FromDecimal(AmbientTemperatureID, in); err == nil {
			t.Errorf("len %d must be rejected", len(in))
		}
	}
}
