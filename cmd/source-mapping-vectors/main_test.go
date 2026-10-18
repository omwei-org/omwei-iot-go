package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRunnerAcceptsPublishedVectors(t *testing.T) {
	root := filepath.Join("..", "..")
	if err := run(filepath.Join(root, "examples", "source-mappings-v1.json"),
		filepath.Join(root, "testdata", "source-mapping-interop-v1.json")); err != nil {
		t.Fatalf("published vectors should pass: %v", err)
	}
}

func TestRunnerFailsOnWireMismatch(t *testing.T) {
	root := filepath.Join("..", "..")
	profiles := filepath.Join(root, "examples", "source-mappings-v1.json")
	raw, err := os.ReadFile(filepath.Join(root, "testdata", "source-mapping-interop-v1.json"))
	if err != nil { t.Fatal(err) }
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil { t.Fatal(err) }
	cases := doc["cases"].([]any)
	first := cases[0].(map[string]any)
	first["expected_wire_hex"] = "ffffffffffff"
	broken, err := json.Marshal(doc)
	if err != nil { t.Fatal(err) }
	vectors := filepath.Join(t.TempDir(), "bad-vectors.json")
	if err := os.WriteFile(vectors, broken, 0600); err != nil { t.Fatal(err) }
	if err := run(profiles, vectors); err == nil {
		t.Fatal("runner must fail when actual wire bytes differ from the vector")
	}
}

func TestRunnerFailsOnUnknownExpectedError(t *testing.T) {
	root := filepath.Join("..", "..")
	profiles := filepath.Join(root, "examples", "source-mappings-v1.json")
	raw, err := os.ReadFile(filepath.Join(root, "testdata", "source-mapping-interop-v1.json"))
	if err != nil { t.Fatal(err) }
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil { t.Fatal(err) }
	cases := doc["cases"].([]any)
	first := cases[0].(map[string]any)
	delete(first, "expected_wire_hex")
	first["expected_error"] = "any-error-is-fine"
	broken, err := json.Marshal(doc)
	if err != nil { t.Fatal(err) }
	vectors := filepath.Join(t.TempDir(), "bad-vectors.json")
	if err := os.WriteFile(vectors, broken, 0600); err != nil { t.Fatal(err) }
	if err := run(profiles, vectors); err == nil {
		t.Fatal("runner must reject unknown expected error names")
	}
}
