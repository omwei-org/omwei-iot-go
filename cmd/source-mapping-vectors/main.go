// source-mapping-vectors runs the published, transport-neutral source-mapping
// interoperability vectors against the Go implementation.
package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	omwei "github.com/omwei-org/omwei-iot-go"
)

type vectorDocument struct {
	SchemaVersion string `json:"schema_version"`
	Cases []vectorCase `json:"cases"`
}

type vectorCase struct {
	Name string `json:"name"`
	Sample omwei.SourceSample `json:"sample"`
	ExpectedWireHex string `json:"expected_wire_hex"`
	ExpectedError string `json:"expected_error"`
}

func expectedErrorName(err error) string {
	switch {
	case errors.Is(err, omwei.ErrSourceIdentityMismatch):
		return "source_identity_mismatch"
	case errors.Is(err, omwei.ErrObservedSourceUnitMismatch):
		return "observed_source_unit_mismatch"
	case errors.Is(err, omwei.ErrInvalidDecimal):
		return "invalid_decimal"
	case errors.Is(err, omwei.ErrOutOfRange):
		return "out_of_range"
	default:
		return ""
	}
}

func run(profilePath, vectorsPath string) error {
	profileBytes, err := os.ReadFile(profilePath)
	if err != nil { return fmt.Errorf("read profiles: %w", err) }
	profiles, err := omwei.ParseSourceMappingDocument(profileBytes)
	if err != nil { return fmt.Errorf("parse profiles: %w", err) }

	vectorBytes, err := os.ReadFile(vectorsPath)
	if err != nil { return fmt.Errorf("read vectors: %w", err) }
	var doc vectorDocument
	if err := json.Unmarshal(vectorBytes, &doc); err != nil { return fmt.Errorf("parse vectors: %w", err) }
	if doc.SchemaVersion != "1" || len(doc.Cases) == 0 {
		return fmt.Errorf("invalid vector document: schema_version must be 1 and cases must not be empty")
	}

	for _, tc := range doc.Cases {
		var mapping *omwei.SourceMapping
		for i := range profiles.Mappings {
			if profiles.Mappings[i].Source == tc.Sample.Source {
				mapping = &profiles.Mappings[i]
				break
			}
		}
		if mapping == nil && tc.ExpectedError == "source_identity_mismatch" {
			for i := range profiles.Mappings {
				candidate := &profiles.Mappings[i]
				if candidate.Source.Protocol == tc.Sample.Source.Protocol &&
					candidate.Source.Device == tc.Sample.Source.Device {
					mapping = candidate
					break
				}
			}
		}
		if mapping == nil {
			return fmt.Errorf("%s: no matching mapping profile", tc.Name)
		}

		result, mapErr := omwei.MapSourceSample(*mapping, tc.Sample)
		if tc.ExpectedError != "" {
			got := expectedErrorName(mapErr)
			if got != tc.ExpectedError {
				return fmt.Errorf("%s: expected error %q, got %q (%v)", tc.Name, tc.ExpectedError, got, mapErr)
			}
			fmt.Printf("PASS %s: %s\n", tc.Name, got)
			continue
		}
		if mapErr != nil {
			return fmt.Errorf("%s: unexpected mapping error: %w", tc.Name, mapErr)
		}
		if tc.ExpectedWireHex == "" {
			return fmt.Errorf("%s: missing expected_wire_hex", tc.Name)
		}
		want, err := hex.DecodeString(tc.ExpectedWireHex)
		if err != nil { return fmt.Errorf("%s: invalid expected wire hex: %w", tc.Name, err) }
		got := omwei.Encode(result.Atom)
		if len(want) != len(got) || string(want) != string(got[:]) {
			return fmt.Errorf("%s: wire mismatch: want %x, got %x", tc.Name, want, got)
		}
		fmt.Printf("PASS %s: %x\n", tc.Name, got)
	}
	fmt.Printf("PASS all %d source-mapping interoperability vectors\n", len(doc.Cases))
	return nil
}

func main() {
	profiles := flag.String("profiles", "examples/source-mappings-v1.json", "source mapping profile JSON")
	vectors := flag.String("vectors", "testdata/source-mapping-interop-v1.json", "interop vector JSON")
	flag.Parse()
	if err := run(*profiles, *vectors); err != nil {
		fmt.Fprintln(os.Stderr, "FAIL:", err)
		os.Exit(1)
	}
}
