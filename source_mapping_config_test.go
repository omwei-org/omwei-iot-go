package omwei

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func validSourceMappingDocument() string {
	return `{
	  "schema_version": "1",
	  "mappings": [
	    {
	      "source": {"protocol":"modbus","device":"inverter-1","point":"holding:40001"},
	      "descriptor_id": 66,
	      "source_unit": "register-tenths-of-degree-C",
	      "canonical_unit": "degree_Celsius",
	      "scale_numerator": 1,
	      "scale_denominator": 10,
	      "offset_numerator": 0,
	      "offset_denominator": 1
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
	doc, err := ParseSourceMappingDocument([]byte(validSourceMappingDocument()))
	if err != nil {
		t.Fatal(err)
	}
	doc.Mappings = append(doc.Mappings, doc.Mappings[0])
	duplicated, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ParseSourceMappingDocument(duplicated)
	if !errors.Is(err, ErrDuplicateSourceIdentity) {
		t.Fatalf("want duplicate source identity, got %v", err)
	}
}
