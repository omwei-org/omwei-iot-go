package omwei

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"testing"
)

type sourceMappingInteropDocument struct {
	SchemaVersion string `json:"schema_version"`
	Cases []struct {
		Name string `json:"name"`
		MappingIndex int `json:"mapping_index"`
		Sample SourceSample `json:"sample"`
		ExpectedWireHex string `json:"expected_wire_hex"`
		ExpectedError string `json:"expected_error"`
	} `json:"cases"`
}

func TestSourceMappingInteropVectors(t *testing.T) {
	profileBytes, err := os.ReadFile("examples/source-mappings-v1.json")
	if err != nil { t.Fatal(err) }
	profiles, err := ParseSourceMappingDocument(profileBytes)
	if err != nil { t.Fatal(err) }

	vectorBytes, err := os.ReadFile("testdata/source-mapping-interop-v1.json")
	if err != nil { t.Fatal(err) }
	var doc sourceMappingInteropDocument
	if err := json.Unmarshal(vectorBytes, &doc); err != nil { t.Fatal(err) }
	if doc.SchemaVersion != "1" || len(doc.Cases) == 0 {
		t.Fatalf("invalid interop vector document: %+v", doc)
	}

	for _, tc := range doc.Cases {
		t.Run(tc.Name, func(t *testing.T) {
			if tc.MappingIndex < 0 || tc.MappingIndex >= len(profiles.Mappings) {
				t.Fatalf("mapping_index %d out of range", tc.MappingIndex)
			}
			result, err := MapSourceSample(profiles.Mappings[tc.MappingIndex], tc.Sample)
			if tc.ExpectedError != "" {
				if err == nil { t.Fatal("expected mapping error, got nil") }
				switch tc.ExpectedError {
				case "source_identity_mismatch":
					if !errors.Is(err, ErrSourceIdentityMismatch) { t.Fatalf("want identity mismatch, got %v", err) }
				case "observed_source_unit_mismatch":
					if !errors.Is(err, ErrObservedSourceUnitMismatch) { t.Fatalf("want unit mismatch, got %v", err) }
				default:
					t.Fatalf("unknown expected_error %q", tc.ExpectedError)
				}
				return
			}
			if err != nil { t.Fatal(err) }
			got := Encode(result.Atom)
			want, err := hex.DecodeString(tc.ExpectedWireHex)
			if err != nil { t.Fatal(err) }
			if string(got[:]) != string(want) {
				t.Fatalf("wire bytes: want %x, got %x", want, got)
			}
		})
	}
}
