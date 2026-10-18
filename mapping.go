package omwei

import "fmt"

// ResolveDescriptor resolves a semantic mapping through the shared registry.
// Vendor-specific field names deliberately do not form part of this API.
func ResolveDescriptor(id uint16) (Descriptor, error) {
	d, ok := Lookup(id)
	if !ok {
		return Descriptor{}, fmt.Errorf("semantic mapping references unknown descriptor: %w", ErrUnknownDescriptor)
	}
	return d, nil
}

// MapCanonicalValue constructs an Atom only after resolving its descriptor
// through the shared registry and validating the canonical encoded value.
func MapCanonicalValue(id uint16, canonicalValue int32) (Atom, error) {
	d, err := ResolveDescriptor(id)
	if err != nil {
		return Atom{}, err
	}
	return NewAtom(d.ID, canonicalValue)
}

// AmbientTemperatureDescriptor returns the registry descriptor used by the
// interoperability example. It does not bypass registry resolution.
func AmbientTemperatureDescriptor() (Descriptor, error) {
	return ResolveDescriptor(AmbientTemperatureID)
}
