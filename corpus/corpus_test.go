package corpus

import (
	"os"
	"strings"
	"testing"
)

const valid = `registry_id: org.omwei.core
schema_version: "1"
corpus_version: "0.1.0"
version: 0.1
descriptors:
  - id: "0x0042"
    namespace: core
    status: active
    version: 1
    label: ambient_temperature
    semantic:
      quantity_kind: temperature
      mappings:
        - vocabulary: QUDT
          concept: Temperature
    unit:
      system: SI
      canonical: degree_Celsius
    encoding:
      datatype: int32
      signed: true
      byte_order: big_endian
      scale: "0.001"
      offset: 0
    constraints:
      min: -80000
      max: 150000
`

func mustFail(t *testing.T, name, yaml, contains string) {
	t.Helper()
	err := Validate(yaml)
	if err == nil || !strings.Contains(err.Error(), contains) {
		t.Errorf("%s: want error containing %q, got %v", name, contains, err)
	}
}

func TestAcceptsValid(t *testing.T) {
	if err := Validate(valid); err != nil {
		t.Fatal(err)
	}
}

func TestParserReturnsMetadataAndDescriptors(t *testing.T) {
	c, err := Parse(valid)
	if err != nil {
		t.Fatal(err)
	}
	d := c.Descriptors[0]
	if c.Metadata != (Metadata{"org.omwei.core", "1", "0.1.0"}) || len(c.Descriptors) != 1 ||
		d != (Descriptor{0x0042, "ambient_temperature", "degree_Celsius", 1000, 0, -80000, 150000}) {
		t.Fatalf("%+v", c)
	}
}

func TestRejections(t *testing.T) {
	for _, id := range []string{"0X0042", "0042", "0x042", "0x00042"} {
		mustFail(t, id, strings.Replace(valid, `id: "0x0042"`, `id: "`+id+`"`, 1), "canonical form")
	}
	mustFail(t, "unquoted id", strings.Replace(valid, `id: "0x0042"`, `id: 0x0042`, 1), "quoted hexadecimal string")
	i := strings.Index(valid, "descriptors:\n")
	header, desc := valid[:i], valid[i+len("descriptors:\n"):]
	mustFail(t, "duplicate", header+"descriptors:\n"+desc+desc, "duplicate descriptor ID")
	mustFail(t, "status", strings.Replace(valid, "status: active", "status: maybe", 1), "status is not supported")
	mustFail(t, "range", strings.Replace(valid, "min: -80000", "min: 200000", 1), "min must be <= max")
	mustFail(t, "zero scale", strings.Replace(valid, `scale: "0.001"`, `scale: "0"`, 1), "integer reciprocal")
	mustFail(t, "approx scale", strings.Replace(valid, `scale: "0.001"`, `scale: "0.3333333333"`, 1), "exact positive integer reciprocal")
	mustFail(t, "unquoted scale", strings.Replace(valid, `scale: "0.001"`, `scale: 0.001`, 1), "quoted base-10 decimal string")
	mustFail(t, "schema", strings.Replace(valid, `schema_version: "1"`, `schema_version: "2"`, 1), "unsupported schema_version")
	mustFail(t, "missing identity", strings.Replace(valid, "registry_id: org.omwei.core\n", "", 1), "registry_id")
	mustFail(t, "signed", strings.Replace(valid, "signed: true", "signed: false", 1), "signed must be true")
	mustFail(t, "datatype", strings.Replace(valid, "datatype: int32", "datatype: int16", 1), "datatype must be int32")
	mustFail(t, "offset", strings.Replace(valid, "offset: 0", "offset: 99999999999", 1), "offset must fit int32")
	mustFail(t, "version", strings.Replace(valid, "    version: 1\n", "    version: 0\n", 1), "version must be a positive integer")
	mustFail(t, "tabs", strings.Replace(valid, "    status: active", "\tstatus: active", 1), "tab indentation")
	mustFail(t, "flow", strings.Replace(valid, "namespace: core", "namespace: {a: b}", 1), "unsupported YAML construct")
}

func TestSemver(t *testing.T) {
	for _, v := range []string{"0.1", "v0.1.0", "0.01.0", "0.1.0-", "0.1.0+"} {
		if err := Validate(strings.Replace(valid, `corpus_version: "0.1.0"`, `corpus_version: "`+v+`"`, 1)); err == nil {
			t.Errorf("%q must be rejected", v)
		}
	}
	if err := Validate(strings.Replace(valid, `corpus_version: "0.1.0"`, `corpus_version: "0.1.0-beta.1+build.7"`, 1)); err != nil {
		t.Error(err)
	}
}

func TestExactScaleExamples(t *testing.T) {
	for in, want := range map[string]int32{"0.001": 1000, "0.25": 4, "1": 1} {
		if got, err := ExactScaleMultiplier(in); err != nil || got != want {
			t.Errorf("%s: %d %v", in, got, err)
		}
	}
	for _, in := range []string{"0.3333333333", "1e-3", "", "-1", "+1", "0", "1.", ".5", "1.2.3", "2"} {
		if _, err := ExactScaleMultiplier(in); err == nil {
			t.Errorf("%q must be rejected", in)
		}
	}
}

func TestShippedCorpusIsValid(t *testing.T) {
	b, err := os.ReadFile("../registry/corpus-v0.1.yaml")
	if err != nil {
		t.Fatal(err)
	}
	c, err := Parse(string(b))
	if err != nil || len(c.Descriptors) != 6 {
		t.Fatalf("%v %+v", err, c)
	}
}
