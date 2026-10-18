package omwei

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
)

func parseID(t *testing.T, s string) uint16 {
	t.Helper()
	n, err := strconv.ParseUint(strings.TrimPrefix(s, "0x"), 16, 16)
	if err != nil {
		t.Fatal(err)
	}
	return uint16(n)
}

func loadVectors(t *testing.T, file string, v any) {
	t.Helper()
	b, err := os.ReadFile("testdata/" + file)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatal(err)
	}
}

func TestAtomVectors(t *testing.T) {
	var set struct {
		VectorSet string `json:"vector_set"`
		Valid     []struct {
			Name, Label, Unit string
			DescriptorID      string `json:"descriptor_id"`
			EncodedValue      int32  `json:"encoded_value"`
			BytesHex          string `json:"bytes_hex"`
		}
		Invalid []struct {
			Name         string
			DescriptorID string `json:"descriptor_id"`
			EncodedValue int32  `json:"encoded_value"`
			Reason       string
		}
	}
	loadVectors(t, "atom-v1.json", &set)
	if set.VectorSet != "omwei-iot-atom-v1" || len(set.Valid) != 18 || len(set.Invalid) != 17 {
		t.Fatalf("unexpected vector set: %s valid=%d invalid=%d", set.VectorSet, len(set.Valid), len(set.Invalid))
	}
	for _, v := range set.Valid {
		id := parseID(t, v.DescriptorID)
		d, ok := Lookup(id)
		if !ok || d.Label != v.Label || d.Unit != v.Unit {
			t.Errorf("%s: descriptor mismatch %+v", v.Name, d)
			continue
		}
		a, err := NewAtom(id, v.EncodedValue)
		if err != nil {
			t.Errorf("%s: %v", v.Name, err)
			continue
		}
		want, _ := hex.DecodeString(v.BytesHex)
		got := Encode(a)
		if !bytes.Equal(got[:], want) {
			t.Errorf("%s: bytes %x want %x", v.Name, got, want)
		}
		if back, err := Decode(want); err != nil || back != a {
			t.Errorf("%s: decode %v %+v", v.Name, err, back)
		}
	}
	for _, v := range set.Invalid {
		_, err := NewAtom(parseID(t, v.DescriptorID), v.EncodedValue)
		switch v.Reason {
		case "unknown_descriptor":
			if !errors.Is(err, ErrUnknownDescriptor) {
				t.Errorf("%s: %v", v.Name, err)
			}
		case "value_out_of_range":
			if !errors.Is(err, ErrOutOfRange) {
				t.Errorf("%s: %v", v.Name, err)
			}
		default:
			t.Errorf("%s: unrecognized reason %s", v.Name, v.Reason)
		}
	}
}

func TestDecimalVectors(t *testing.T) {
	var set struct {
		VectorSet string `json:"vector_set"`
		Valid     []struct {
			Name         string
			DescriptorID string `json:"descriptor_id"`
			Input        string
			EncodedValue int32 `json:"encoded_value"`
		}
		Invalid []struct {
			Name         string
			DescriptorID string `json:"descriptor_id"`
			Input        string
			Reason       string
		}
	}
	loadVectors(t, "decimal-conversion-v1.json", &set)
	if set.VectorSet != "omwei-iot-decimal-conversion-v1" || len(set.Valid) != 13 || len(set.Invalid) != 16 {
		t.Fatalf("unexpected vector set: %s valid=%d invalid=%d", set.VectorSet, len(set.Valid), len(set.Invalid))
	}
	for _, v := range set.Valid {
		a, err := FromDecimal(parseID(t, v.DescriptorID), v.Input)
		if err != nil || a.CanonicalValue != v.EncodedValue {
			t.Errorf("%s: %+v %v", v.Name, a, err)
		}
	}
	sentinel := map[string]error{
		"invalid_decimal": ErrInvalidDecimal, "unknown_descriptor": ErrUnknownDescriptor, "value_out_of_range": ErrOutOfRange,
	}
	for _, v := range set.Invalid {
		_, err := FromDecimal(parseID(t, v.DescriptorID), v.Input)
		if want, ok := sentinel[v.Reason]; !ok || !errors.Is(err, want) {
			t.Errorf("%s: want %s, got %v", v.Name, v.Reason, err)
		}
	}
}
