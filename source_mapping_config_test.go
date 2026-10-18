package omwei

import (
	"errors"
	"strings"
	"testing"
)

func validSourceMappingDocument() string {
	return `{
	  "schema_version": "1",
	  "mappings": [
	    {
	      "Source": {"Protocol":"modbus","Device":"inverter-1","Point":"holding:40001"},
	      "DescriptorID": 66,
	      "SourceUnit": "register-tenths-of-degree-C",
	      "CanonicalUnit": "degree_Celsius",
	      "ScaleNumerator": 1,
	      "ScaleDenominator": 10,
	      "OffsetNumerator": 0,
	      "OffsetDenominator": 1
	    }
	  ]
	}`
}

func TestParseSourceMappingDocument(t *testing.T) {
	doc, err := ParseSourceMappingDocument([]byte(validSourceMappingDocument()))
	if err != nil {
		t.Fatal(err)
	}
	if doc.SchemaVersion != "1" || len(doc.Mappings) != 1 || doc.Mappings[0].DescriptorID != AmbientTemperatureID {
		t.Fatalf("unexpected document: %+v", doc)
	}
}

func TestParseSourceMappingDocumentRejectsInvalidDocuments(t *testing.T) {
	tests := []struct {
		name, input string
		want        error
	}{
		{"unknown schema", strings.Replace(validSourceMappingDocument(), `"schema_version": "1"`, `"schema_version": "2"`, 1), ErrInvalidSourceMappingDocument},
		{"unknown field", strings.Replace(validSourceMappingDocument(), `"schema_version": "1"`, `"schema_version": "1", "extra": true`, 1), ErrInvalidSourceMappingDocument},
		{"trailing value", validSourceMappingDocument() + ` {}`, ErrInvalidSourceMappingDocument},
		{"wrong canonical unit", strings.Replace(validSourceMappingDocument(), `"degree_Celsius"`, `"degree_Fahrenheit"`, 1), ErrInvalidSourceMappingDocument},
		{"duplicate source", strings.Replace(validSourceMappingDocument(), `"mappings": [`, `"mappings": [`+strings.TrimSpace(strings.Split(validSourceMappingDocument(), `"mappings": [`)[1])[:0], 1), ErrInvalidSourceMappingDocument},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseSourceMappingDocument([]byte(tt.input))
			if !errors.Is(err, tt.want) {
				t.Fatalf("want %v, got %v", tt.want, err)
			}
		})
	}
}

func TestParseSourceMappingDocumentRejectsDuplicateSourceIdentity(t *testing.T) {
	one := strings.TrimSpace(validSourceMappingDocument())
	entryStart := strings.Index(one, `    {`)
	entryEnd := strings.Index(one[entryStart:], `    }`) + entryStart + len("    }")
	if entryStart < 0 || entryEnd < len("    }") {
		t.Fatal("could not isolate fixture mapping")
	}
	entry := one[entryStart:entryEnd]
	duplicated := strings.Replace(one, entry, entry+",\n"+entry, 1)
	_, err := ParseSourceMappingDocument([]byte(duplicated))
	if !errors.Is(err, ErrDuplicateSourceIdentity) {
		t.Fatalf("want duplicate source identity, got %v", err)
	}
}
