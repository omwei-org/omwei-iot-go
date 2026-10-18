package omwei

import (
	"bytes"
	"errors"
	"testing"
)

func TestRegistryResolvesTemperature(t *testing.T) {
	d, ok := Lookup(AmbientTemperatureID)
	if !ok || d.Label != "ambient_temperature" || d.Unit != "degree_Celsius" || d.Scale != 1000 {
		t.Fatalf("unexpected descriptor %+v ok=%v", d, ok)
	}
}

func TestCompiledCorpusIdentity(t *testing.T) {
	if RegistryID != "org.omwei.core" || SchemaVersion != "1" || CorpusVersion != "0.1.0" {
		t.Fatal(RegistryID, SchemaVersion, CorpusVersion)
	}
}

func TestDeterministicBigEndianEncoding(t *testing.T) {
	a, err := NewAtom(AmbientTemperatureID, 22_500)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{0x00, 0x42, 0x00, 0x00, 0x57, 0xE4}
	got := Encode(a)
	if !bytes.Equal(got[:], want) {
		t.Fatalf("got %x want %x", got, want)
	}
	back, err := Decode(got[:])
	if err != nil || back != a {
		t.Fatalf("round trip: %v %+v", err, back)
	}
}

func TestEncode128IsPaddedSixBytes(t *testing.T) {
	a, _ := NewAtom(AmbientTemperatureID, 22_500)
	e, e128 := Encode(a), Encode128(a)
	if len(e128) != 16 || !bytes.Equal(e128[:6], e[:]) || !bytes.Equal(e128[6:], make([]byte, 10)) {
		t.Fatalf("%x", e128)
	}
}

func TestUnknownDescriptorRejected(t *testing.T) {
	if _, err := NewAtom(0xFFFF, 1); !errors.Is(err, ErrUnknownDescriptor) {
		t.Fatal(err)
	}
	if _, ok := Lookup(0xFFFF); ok {
		t.Fatal("0xFFFF must not resolve")
	}
}

func TestMalformedAtomLengths(t *testing.T) {
	for _, b := range [][]byte{nil, {0x00, 0x42, 0x00, 0x00, 0x57}, {0x00, 0x42, 0x00, 0x00, 0x57, 0xE4, 0x00}} {
		if _, err := Decode(b); !errors.Is(err, ErrInvalidAtomLength) {
			t.Fatalf("%x: %v", b, err)
		}
	}
}
