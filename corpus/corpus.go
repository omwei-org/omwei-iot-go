// Package corpus validates and parses OMWEI registry corpora (the YAML source
// of truth shared with the Rust implementation). It is stdlib-only and is used
// by the code generator and by tooling; the runtime package does not import it.
package corpus

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
)

// Metadata identifies a corpus.
type Metadata struct {
	RegistryID    string
	SchemaVersion string
	CorpusVersion string
}

// Descriptor is one validated registry entry in its runtime (integer) form.
type Descriptor struct {
	ID     uint16
	Label  string
	Unit   string
	Scale  int32 // integer multiplier: encoded = round((physical - offset) * Scale)
	Offset int32
	Min    int32
	Max    int32
}

// Corpus is a validated corpus.
type Corpus struct {
	Metadata    Metadata
	Descriptors []Descriptor
}

// Parse validates yaml with the same rules as the Rust build script and returns
// the descriptors in document order.
func Parse(yaml string) (*Corpus, error) {
	root, err := validate(yaml)
	if err != nil {
		return nil, err
	}
	str := func(m *Map, k string) string { v, _ := m.Get(k); s, _ := v.(Scalar).Str(); return s }
	c := &Corpus{Metadata: Metadata{
		RegistryID:    str(root, "registry_id"),
		SchemaVersion: str(root, "schema_version"),
		CorpusVersion: str(root, "corpus_version"),
	}}
	dv, _ := root.Get("descriptors")
	for _, item := range dv.([]any) {
		d := item.(*Map)
		sub := func(k string) *Map { v, _ := d.Get(k); return v.(*Map) }
		i64 := func(m *Map, k string) int32 { v, _ := m.Get(k); n, _ := v.(Scalar).Int64(); return int32(n) }
		hex := str(d, "id")
		var id uint16
		fmt.Sscanf(hex[2:], "%x", &id) // validated: exactly 4 hex digits
		enc, cons := sub("encoding"), sub("constraints")
		scale, err := ExactScaleMultiplier(str(enc, "scale"))
		if err != nil {
			return nil, fmt.Errorf("descriptor 0x%04X scale %w", id, err)
		}
		c.Descriptors = append(c.Descriptors, Descriptor{
			ID: id, Label: str(d, "label"), Unit: str(sub("unit"), "canonical"),
			Scale: scale, Offset: i64(enc, "offset"), Min: i64(cons, "min"), Max: i64(cons, "max"),
		})
	}
	return c, nil
}

// Validate checks a corpus deterministically, without network access.
func Validate(yaml string) error { _, err := validate(yaml); return err }

func validate(yaml string) (*Map, error) {
	rootAny, err := parseYAML(yaml)
	if err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}
	root, ok := rootAny.(*Map)
	if !ok {
		return nil, errors.New("corpus root must be a mapping")
	}
	if _, err := nonempty(root, "registry_id"); err != nil {
		return nil, err
	}
	schema, err := nonempty(root, "schema_version")
	if err != nil {
		return nil, err
	}
	if schema != "1" {
		return nil, fmt.Errorf("unsupported schema_version: %s", schema)
	}
	cv, err := nonempty(root, "corpus_version")
	if err != nil {
		return nil, err
	}
	if !isSemver(cv) {
		return nil, fmt.Errorf("corpus_version must be MAJOR.MINOR.PATCH: %s", cv)
	}
	dv, _ := root.Get("descriptors")
	descs, ok := dv.([]any)
	if !ok {
		return nil, errors.New("descriptors must be a sequence")
	}
	if len(descs) == 0 {
		return nil, errors.New("descriptors must not be empty")
	}
	ids := map[uint16]bool{}
	for i, item := range descs {
		if err := validateDescriptor(item, i, ids); err != nil {
			return nil, err
		}
	}
	return root, nil
}

func validateDescriptor(item any, i int, ids map[uint16]bool) error {
	p := fmt.Sprintf("descriptors[%d]", i)
	d, ok := item.(*Map)
	if !ok {
		return fmt.Errorf("%s must be a mapping", p)
	}
	idv, ok := d.Get("id")
	if !ok {
		return fmt.Errorf("%s.id is required", p)
	}
	idtext, ok := scalarStr(idv)
	if !ok {
		return fmt.Errorf(`%s.id must be a quoted hexadecimal string such as "0x0042"`, p)
	}
	hex, ok := strings.CutPrefix(idtext, "0x")
	if !ok || len(hex) != 4 || !isHex(hex) {
		return fmt.Errorf(`%s.id must use canonical form "0xNNNN"`, p)
	}
	var id uint16
	fmt.Sscanf(hex, "%x", &id)
	if ids[id] {
		return fmt.Errorf("duplicate descriptor ID: 0x%04X", id)
	}
	ids[id] = true

	ns, err := descString(d, "namespace", p)
	if err != nil {
		return err
	}
	if !contains([]string{"core", "industry", "vendor", "private"}, ns) {
		return fmt.Errorf("%s.namespace is not supported: %s", p, ns)
	}
	st, err := descString(d, "status", p)
	if err != nil {
		return err
	}
	if !contains([]string{"active", "deprecated", "retired"}, st) {
		return fmt.Errorf("%s.status is not supported: %s", p, st)
	}
	if _, err := descString(d, "label", p); err != nil {
		return err
	}
	sem, err := childMap(d, "semantic", p)
	if err != nil {
		return err
	}
	if _, err := nonempty(sem, "quantity_kind"); err != nil {
		return fmt.Errorf("%s.%w", p, err)
	}
	if mv, ok := sem.Get("mappings"); ok {
		ms, ok := mv.([]any)
		if !ok {
			return fmt.Errorf("%s.semantic.mappings must be a sequence", p)
		}
		for j, mi := range ms {
			m, ok := mi.(*Map)
			if !ok {
				return fmt.Errorf("%s.semantic.mappings[%d] must be a mapping", p, j)
			}
			for _, key := range []string{"vocabulary", "concept"} {
				v, _ := m.Get(key)
				if s, ok := scalarStr(v); !ok || strings.TrimSpace(s) == "" {
					return fmt.Errorf("%s.semantic.mappings[%d].%s is required", p, j, key)
				}
			}
		}
	}
	unit, err := childMap(d, "unit", p)
	if err != nil {
		return err
	}
	for _, k := range []string{"system", "canonical"} {
		if _, err := nonempty(unit, k); err != nil {
			return fmt.Errorf("%s.%w", p, err)
		}
	}
	enc, err := childMap(d, "encoding", p)
	if err != nil {
		return err
	}
	for _, kv := range [][2]string{{"datatype", "int32"}, {"byte_order", "big_endian"}} {
		v, _ := enc.Get(kv[0])
		actual, ok := scalarStr(v)
		if !ok {
			return fmt.Errorf("%s.encoding.%s is required", p, kv[0])
		}
		if actual != kv[1] {
			return fmt.Errorf("%s.encoding.%s must be %s", p, kv[0], kv[1])
		}
	}
	sv, _ := enc.Get("signed")
	if b, ok := sv.(Scalar); !ok || b.Kind != KindBool {
		return fmt.Errorf("%s.encoding.signed must be true for int32", p)
	} else if v, _ := b.Bool(); !v {
		return fmt.Errorf("%s.encoding.signed must be true for int32", p)
	}
	sc, _ := enc.Get("scale")
	scaleText, ok := scalarStr(sc)
	if !ok {
		return fmt.Errorf("%s.encoding.scale must be a quoted base-10 decimal string", p)
	}
	if _, err := ExactScaleMultiplier(scaleText); err != nil {
		return fmt.Errorf("%s.encoding.scale %w", p, err)
	}
	ov, _ := enc.Get("offset")
	if n, ok := scalarInt(ov); !ok || n < math.MinInt32 || n > math.MaxInt32 {
		return fmt.Errorf("%s.encoding.offset must fit int32", p)
	}
	c, err := childMap(d, "constraints", p)
	if err != nil {
		return err
	}
	min, err := int32Field(c, "min", p)
	if err != nil {
		return err
	}
	max, err := int32Field(c, "max", p)
	if err != nil {
		return err
	}
	if min > max {
		return fmt.Errorf("%s.constraints.min must be <= max", p)
	}
	vv, _ := d.Get("version")
	if n, ok := scalarInt(vv); !ok || n <= 0 {
		return fmt.Errorf("%s.version must be a positive integer", p)
	}
	return nil
}

var maxU128 = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1))

// ExactScaleMultiplier returns the exact positive integer R for a decimal scale
// where 1/scale = R. No floating point and no tolerance is used; exponents are
// rejected so the corpus has a single auditable decimal representation.
func ExactScaleMultiplier(scale string) (int32, error) {
	bad := errors.New("must be a positive plain decimal with an integer reciprocal")
	if scale == "" || scale[0] == '-' || scale[0] == '+' {
		return 0, bad
	}
	whole, fraction, hasDot := strings.Cut(scale, ".")
	if strings.Contains(fraction, ".") || whole == "" || !isDigits(whole) || !isDigits(fraction) || (hasDot && fraction == "") {
		return 0, bad
	}
	num, _ := new(big.Int).SetString(whole+fraction, 10)
	if num.Cmp(maxU128) > 0 {
		return 0, errors.New("is too large for exact decimal validation")
	}
	if len(fraction) > 38 { // 10^39 overflows u128
		return 0, errors.New("has excessive decimal precision")
	}
	den := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(len(fraction))), nil)
	if num.Sign() == 0 || new(big.Int).Mod(den, num).Sign() != 0 {
		return 0, errors.New("must have an exact positive integer reciprocal")
	}
	r := new(big.Int).Div(den, num)
	if r.Sign() == 0 || r.Cmp(big.NewInt(math.MaxInt32)) > 0 {
		return 0, errors.New("integer reciprocal must fit a positive int32")
	}
	return int32(r.Int64()), nil
}

func isSemver(v string) bool {
	core := v
	var pre, build string
	var hasPre, hasBuild bool
	if l, r, ok := strings.Cut(v, "+"); ok {
		if r == "" || strings.Contains(r, "+") {
			return false
		}
		core, build, hasBuild = l, r, true
	}
	if l, r, ok := strings.Cut(core, "-"); ok {
		if r == "" {
			return false
		}
		core, pre, hasPre = l, r, true
	}
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return false
	}
	for _, p := range parts {
		if p == "" || !isDigits(p) || (p != "0" && strings.HasPrefix(p, "0")) {
			return false
		}
	}
	ident := func(s string, prerelease bool) bool {
		for _, id := range strings.Split(s, ".") {
			if id == "" {
				return false
			}
			for _, c := range id {
				if !(c == '-' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z') {
					return false
				}
			}
			if prerelease && isDigits(id) && len(id) > 1 && id[0] == '0' {
				return false
			}
		}
		return true
	}
	return (!hasPre || ident(pre, true)) && (!hasBuild || ident(build, false))
}

func isDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func isHex(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return false
		}
	}
	return true
}

func contains(set []string, s string) bool {
	for _, x := range set {
		if x == s {
			return true
		}
	}
	return false
}

func scalarStr(v any) (string, bool) {
	s, ok := v.(Scalar)
	if !ok {
		return "", false
	}
	return s.Str()
}

func scalarInt(v any) (int64, bool) {
	s, ok := v.(Scalar)
	if !ok {
		return 0, false
	}
	return s.Int64()
}

func nonempty(m *Map, key string) (string, error) {
	v, _ := m.Get(key)
	if s, ok := scalarStr(v); ok && strings.TrimSpace(s) != "" {
		return s, nil
	}
	return "", fmt.Errorf("%s must be a non-empty string", key)
}

func descString(m *Map, key, p string) (string, error) {
	s, err := nonempty(m, key)
	if err != nil {
		return "", fmt.Errorf("%s.%w", p, err)
	}
	return s, nil
}

func childMap(m *Map, key, p string) (*Map, error) {
	v, _ := m.Get(key)
	if c, ok := v.(*Map); ok {
		return c, nil
	}
	return nil, fmt.Errorf("%s.%s is required", p, key)
}

func int32Field(m *Map, key, p string) (int32, error) {
	v, _ := m.Get(key)
	if n, ok := scalarInt(v); ok && n >= math.MinInt32 && n <= math.MaxInt32 {
		return int32(n), nil
	}
	return 0, fmt.Errorf("%s.constraints.%s must fit int32", p, key)
}
