package omwei

import (
	"errors"
	"testing"
)

func TestResolveDescriptorUsesSharedRegistry(t *testing.T) {
	d, err := ResolveDescriptor(AmbientTemperatureID)
	if err != nil {
		t.Fatal(err)
	}
	if d.Label != "ambient_temperature" || d.Unit != "degree_Celsius" || d.Scale != 1000 {
		t.Fatalf("unexpected descriptor: %+v", d)
	}
}

func TestResolveDescriptorRejectsUnknownID(t *testing.T) {
	_, err := ResolveDescriptor(0xffff)
	if !errors.Is(err, ErrUnknownDescriptor) {
		t.Fatalf("want ErrUnknownDescriptor, got %v", err)
	}
}

func TestMapCanonicalValueValidatesRange(t *testing.T) {
	a, err := MapCanonicalValue(AmbientTemperatureID, 22500)
	if err != nil {
		t.Fatal(err)
	}
	if a != (Atom{DescriptorID: AmbientTemperatureID, CanonicalValue: 22500}) {
		t.Fatalf("unexpected atom: %+v", a)
	}
	if _, err := MapCanonicalValue(AmbientTemperatureID, 150001); !errors.Is(err, ErrOutOfRange) {
		t.Fatalf("want ErrOutOfRange, got %v", err)
	}
}

func TestAmbientTemperatureDescriptorMatchesRegistry(t *testing.T) {
	d, err := AmbientTemperatureDescriptor()
	if err != nil {
		t.Fatal(err)
	}
	registry, ok := Lookup(AmbientTemperatureID)
	if !ok || d != registry {
		t.Fatalf("mapping descriptor %+v does not match registry %+v", d, registry)
	}
}
