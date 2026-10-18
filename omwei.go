// Package omwei implements OMWEI semantic interoperability primitives for
// heterogeneous IoT systems: a registry of semantic descriptors, a validated
// 6-byte big-endian semantic atom, and deterministic decimal conversion.
//
// It is a Go port of the Rust crate omwei-iot. Both implementations are driven
// by the same YAML corpus and the same conformance vectors. The package has no
// dependencies outside the standard library and does not use cgo.
package omwei

import (
	"encoding/binary"
	"errors"
	"strings"
)

//go:generate go run ./cmd/omwei-gen -in registry/corpus-v0.1.yaml -out registry_generated.go

// AmbientTemperatureID is the descriptor ID of ambient_temperature.
const AmbientTemperatureID uint16 = 0x0042

// SemanticMapping identifies an exact concept in an external vocabulary.
type SemanticMapping struct {
	Vocabulary string
	Concept    string
}

// Descriptor is one compiled-in registry entry.
type Descriptor struct {
	ID           uint16
	Label        string
	Unit         string // canonical unit
	QuantityKind string
	Mappings     []SemanticMapping
	Scale        int32 // integer multiplier: encoded = round((physical - Offset) * Scale)
	Offset       int32
	Min          int32 // inclusive bounds on the encoded value
	Max          int32
}

// Errors returned by the package. Use errors.Is to test for them.
var (
	ErrUnknownDescriptor        = errors.New("unknown descriptor")
	ErrUnknownExternalMapping   = errors.New("unknown external mapping")
	ErrAmbiguousExternalMapping = errors.New("ambiguous external mapping")
	ErrInvalidSourceMapping     = errors.New("invalid source mapping")
	ErrSourceUnitMismatch       = errors.New("source transform target unit does not match descriptor unit")
	ErrOutOfRange               = errors.New("value outside descriptor constraints")
	ErrInvalidAtomLength        = errors.New("invalid OMWEI atom length")
	ErrInvalidDecimal           = errors.New("invalid decimal")
)

// cloneDescriptor prevents callers from mutating the compiled-in registry through
// the descriptor's slice-valued semantic mappings.
func cloneDescriptor(d Descriptor) Descriptor {
	d.Mappings = append([]SemanticMapping(nil), d.Mappings...)
	return d
}

// Lookup resolves a descriptor ID against the compiled-in corpus.
func Lookup(id uint16) (Descriptor, bool) {
	for _, d := range descriptors {
		if d.ID == id {
			return cloneDescriptor(d), true
		}
	}
	return Descriptor{}, false
}

// Descriptors returns a copy of the compiled-in corpus in registry order.
func Descriptors() []Descriptor {
	out := make([]Descriptor, len(descriptors))
	for i, d := range descriptors {
		out[i] = cloneDescriptor(d)
	}
	return out
}

// LookupExternal resolves an exact vocabulary/concept pair; it never infers
// equivalence or normalizes identifiers across vocabularies.
func LookupExternal(vocabulary, concept string) (Descriptor, error) {
	if strings.TrimSpace(vocabulary) == "" || strings.TrimSpace(concept) == "" {
		return Descriptor{}, ErrUnknownExternalMapping
	}
	var found Descriptor
	matched := false
	for _, d := range descriptors {
		for _, m := range d.Mappings {
			if m.Vocabulary == vocabulary && m.Concept == concept {
				if matched && found.ID != d.ID {
					return Descriptor{}, ErrAmbiguousExternalMapping
				}
				found, matched = d, true
			}
		}
	}
	if !matched {
		return Descriptor{}, ErrUnknownExternalMapping
	}
	return cloneDescriptor(found), nil
}

// Atom is a descriptor ID paired with a value in the descriptor's encoded
// integer domain. Construct it with NewAtom so the constraints are enforced.
type Atom struct {
	DescriptorID   uint16
	CanonicalValue int32
}

// AtomSize is the wire size of an encoded atom in bytes.
const AtomSize = 6

// NewAtom validates canonicalValue against the descriptor's constraints.
func NewAtom(descriptorID uint16, canonicalValue int32) (Atom, error) {
	d, ok := Lookup(descriptorID)
	if !ok {
		return Atom{}, ErrUnknownDescriptor
	}
	if canonicalValue < d.Min || canonicalValue > d.Max {
		return Atom{}, ErrOutOfRange
	}
	return Atom{descriptorID, canonicalValue}, nil
}

// Encode returns the 6-byte wire form: uint16 descriptor ID then int32 value,
// both big-endian.
func Encode(a Atom) [AtomSize]byte {
	var out [AtomSize]byte
	binary.BigEndian.PutUint16(out[:2], a.DescriptorID)
	binary.BigEndian.PutUint32(out[2:], uint32(a.CanonicalValue))
	return out
}

// Encode128 returns the atom zero-padded to the historical 128-bit target.
func Encode128(a Atom) [16]byte {
	var out [16]byte
	e := Encode(a)
	copy(out[:], e[:])
	return out
}

// Decode parses and validates a 6-byte atom.
func Decode(b []byte) (Atom, error) {
	if len(b) != AtomSize {
		return Atom{}, ErrInvalidAtomLength
	}
	return NewAtom(binary.BigEndian.Uint16(b[:2]), int32(binary.BigEndian.Uint32(b[2:])))
}
