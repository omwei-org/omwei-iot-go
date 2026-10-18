package omwei

import (
	"os"
	"testing"

	"github.com/omwei-org/omwei-iot-go/corpus"
)

// TestGeneratedTableMatchesCorpus fails when registry_generated.go has drifted
// from the YAML source of truth (run `go generate ./...` to fix).
func TestGeneratedTableMatchesCorpus(t *testing.T) {
	src, err := os.ReadFile("registry/corpus-v0.1.yaml")
	if err != nil {
		t.Fatal(err)
	}
	c, err := corpus.Parse(string(src))
	if err != nil {
		t.Fatal(err)
	}
	if c.Metadata.RegistryID != RegistryID || c.Metadata.SchemaVersion != SchemaVersion || c.Metadata.CorpusVersion != CorpusVersion {
		t.Fatalf("metadata drift: %+v", c.Metadata)
	}
	got := Descriptors()
	if len(got) != len(c.Descriptors) {
		t.Fatalf("descriptor count drift: %d vs %d", len(got), len(c.Descriptors))
	}
	for i, w := range c.Descriptors {
		if g := got[i]; g.ID != w.ID || g.Label != w.Label || g.Unit != w.Unit || g.Scale != w.Scale || g.Offset != w.Offset || g.Min != w.Min || g.Max != w.Max {
			t.Errorf("drift at %d: %+v vs %+v", i, g, w)
		}
	}
}
