package omwei

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const SourceMappingSchemaVersion = "1"

var (
	ErrInvalidSourceMappingDocument = errors.New("invalid source mapping document")
	ErrDuplicateSourceIdentity      = errors.New("duplicate source identity")
)

// SourceMappingDocument is the versioned, reviewable JSON representation of
// source profiles. It deliberately does not imply any protocol-specific schema.
type SourceMappingDocument struct {
	SchemaVersion string          `json:"schema_version"`
	Mappings      []SourceMapping `json:"mappings"`
}

// ParseSourceMappingDocument decodes one strict JSON document and validates
// every profile against the compiled-in descriptor registry.
func ParseSourceMappingDocument(data []byte) (SourceMappingDocument, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var doc SourceMappingDocument
	if err := dec.Decode(&doc); err != nil {
		return SourceMappingDocument{}, fmt.Errorf("%w: %v", ErrInvalidSourceMappingDocument, err)
	}
	var trailing any
	if err := dec.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("multiple JSON values")
		}
		return SourceMappingDocument{}, fmt.Errorf("%w: %v", ErrInvalidSourceMappingDocument, err)
	}
	if doc.SchemaVersion != SourceMappingSchemaVersion {
		return SourceMappingDocument{}, fmt.Errorf("%w: schema_version must be %q", ErrInvalidSourceMappingDocument, SourceMappingSchemaVersion)
	}
	if len(doc.Mappings) == 0 {
		return SourceMappingDocument{}, fmt.Errorf("%w: mappings must not be empty", ErrInvalidSourceMappingDocument)
	}
	seen := make(map[SourceIdentity]struct{}, len(doc.Mappings))
	for i, m := range doc.Mappings {
		if err := ValidateSourceMapping(m); err != nil {
			return SourceMappingDocument{}, fmt.Errorf("%w: mappings[%d]: %v", ErrInvalidSourceMappingDocument, i, err)
		}
		if _, ok := seen[m.Source]; ok {
			return SourceMappingDocument{}, fmt.Errorf("%w: mappings[%d] identifies %s/%s/%s", ErrDuplicateSourceIdentity, i, m.Source.Protocol, m.Source.Device, m.Source.Point)
		}
		seen[m.Source] = struct{}{}
	}
	return doc, nil
}

// ValidateSourceMapping checks profile completeness and its canonical unit
// against the selected descriptor. SourceUnit is retained as explicit metadata;
// physical unit conversion is never inferred from its label.
func ValidateSourceMapping(mapping SourceMapping) error {
	if strings.TrimSpace(mapping.Source.Protocol) == "" ||
		strings.TrimSpace(mapping.Source.Device) == "" ||
		strings.TrimSpace(mapping.Source.Point) == "" ||
		strings.TrimSpace(mapping.SourceUnit) == "" ||
		strings.TrimSpace(mapping.CanonicalUnit) == "" ||
		mapping.ScaleNumerator == 0 || mapping.ScaleDenominator <= 0 ||
		mapping.OffsetDenominator <= 0 {
		return ErrInvalidSourceMapping
	}
	if _, err := parseDecimal(fmt.Sprintf("%d", mapping.OffsetNumerator)); err != nil {
		return ErrInvalidSourceMapping
	}
	d, ok := Lookup(mapping.DescriptorID)
	if !ok {
		return fmt.Errorf("descriptor 0x%04X: %w", mapping.DescriptorID, ErrUnknownDescriptor)
	}
	if mapping.CanonicalUnit != d.Unit {
		return fmt.Errorf("%w: got %q, want %q", ErrSourceUnitMismatch, mapping.CanonicalUnit, d.Unit)
	}
	return nil
}
