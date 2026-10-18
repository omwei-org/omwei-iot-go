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
)

//go:generate go run ./cmd/omwei-gen -in registry/corpus-v0.1.yaml -out registry_generated.go

// AmbientTemperatureID is the descriptor ID of ambient_temperature.
const AmbientTemperatureID uint16 = 0x0042

// Descriptor is one compiled-in registry entry.
type Descriptor struct {
	ID     uint16
	Label  string
	Unit   string // canonical unit
	Scale  int32  // integer multiplier: encoded = round((physical - Offset) * Scale)
	Offset int32
	Min    int32 // inclusive bounds on the encoded value
	Max    int32
}

// Errors returned by the package. Use errors.Is to test for them.
var (
	ErrUnknownDescriptor = errors.New("unknown descriptor")
	ErrOutOfRange        = errors.New("value outside descriptor constraints")
	ErrInvalidAtomLength = errors.New("invalid OMWEI atom length")
	ErrInvalidDecimal    = errors.New("invalid decimal")
)

// Lookup resolves a descriptor ID against the compiled-in corpus.
func Lookup(id uint16) (Descriptor, bool) {
	for _, d := range descriptors {
		if d.ID == id {
			return d, true
		}
	}
	return Descriptor{}, false
}

// Descriptors returns a copy of the compiled-in corpus in registry order.
func Descriptors() []Descriptor { return append([]Descriptor(nil), descriptors[:]...) }

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
