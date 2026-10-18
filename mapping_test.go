package omwei

import (
	"errors"
	"reflect"
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
	if !ok || !reflect.DeepEqual(d, registry) {
		t.Fatalf("mapping descriptor %+v does not match registry %+v", d, registry)
	}
}


func TestLookupExternalExactVocabularyAndConcept(t *testing.T) {
	d, err := LookupExternal("QUDT", "Temperature")
	if err != nil {
		t.Fatal(err)
	}
	if d.ID != AmbientTemperatureID || d.QuantityKind != "temperature" {
		t.Fatalf("unexpected external lookup result: %+v", d)
	}
	if _, err := LookupExternal("qudt", "Temperature"); !errors.Is(err, ErrUnknownExternalMapping) {
		t.Fatalf("lookup must be exact; got %v", err)
	}
	if _, err := LookupExternal("QUDT", "NotAConcept"); !errors.Is(err, ErrUnknownExternalMapping) {
		t.Fatalf("want unknown external mapping, got %v", err)
	}
}

func TestLookupReturnsIndependentSemanticMappings(t *testing.T) {
	d, ok := Lookup(AmbientTemperatureID)
	if !ok {
		t.Fatal("temperature descriptor missing")
	}
	d.Mappings[0].Concept = "Corrupted"
	again, ok := Lookup(AmbientTemperatureID)
	if !ok || again.Mappings[0].Concept != "Temperature" {
		t.Fatalf("caller mutation escaped into registry: %+v", again)
	}
}
