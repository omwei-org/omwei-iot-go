package corpus

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// This file implements a deliberately small, strict YAML subset parser: block
// mappings, block sequences and scalars (plain, "double" and 'single' quoted).
// It exists so the library and its generator have zero third-party
// dependencies. Anything outside the subset (flow collections, anchors, tags,
// block scalars, tabs) is rejected instead of being guessed at.

// Kind classifies a scalar the way a YAML 1.2 core-schema parser would, which
// matters because the corpus rules distinguish quoted strings from numbers.
type Kind int

const (
	KindNull Kind = iota
	KindBool
	KindInt
	KindFloat
	KindString
)

// Scalar is a parsed YAML scalar.
type Scalar struct {
	Text string
	Kind Kind
}

// Str returns the text when the scalar is a string (quoted, or plain and not
// resolvable to another core type).
func (s Scalar) Str() (string, bool) { return s.Text, s.Kind == KindString }

// Bool returns the value when the scalar is a boolean.
func (s Scalar) Bool() (bool, bool) {
	if s.Kind != KindBool {
		return false, false
	}
	return strings.EqualFold(s.Text, "true"), true
}

// Int64 returns the value when the scalar is an integer that fits int64.
func (s Scalar) Int64() (int64, bool) {
	if s.Kind != KindInt {
		return 0, false
	}
	t := s.Text
	neg := false
	switch {
	case strings.HasPrefix(t, "-"):
		neg, t = true, t[1:]
	case strings.HasPrefix(t, "+"):
		t = t[1:]
	}
	base := 10
	switch {
	case strings.HasPrefix(t, "0x"):
		base, t = 16, t[2:]
	case strings.HasPrefix(t, "0o"):
		base, t = 8, t[2:]
	}
	u, err := strconv.ParseUint(t, base, 64)
	if err != nil {
		return 0, false
	}
	if neg {
		if u > 1<<63 {
			return 0, false
		}
		return -int64(u), true
	}
	if u > 1<<63-1 {
		return 0, false
	}
	return int64(u), true
}

// Map is an insertion-ordered mapping with string keys.
type Map struct {
	keys []string
	vals map[string]any
}

// Get returns the value stored under key.
func (m *Map) Get(key string) (any, bool) { v, ok := m.vals[key]; return v, ok }

// Keys returns the keys in document order.
func (m *Map) Keys() []string { return m.keys }

type line struct {
	indent int
	text   string
	no     int
}

type parser struct {
	lines []line
	pos   int
}

var (
	reInt   = regexp.MustCompile(`^([-+]?[0-9]+|0x[0-9a-fA-F]+|0o[0-7]+)$`)
	reFloat = regexp.MustCompile(`^([-+]?(\.[0-9]+|[0-9]+(\.[0-9]*)?)([eE][-+]?[0-9]+)?|[-+]?\.(inf|Inf|INF)|\.(nan|NaN|NAN))$`)
)

func errAt(no int, format string, a ...any) error {
	return fmt.Errorf("yaml line %d: %s", no, fmt.Sprintf(format, a...))
}

// parseYAML parses src into *Map, []any or Scalar.
func parseYAML(src string) (any, error) {
	var lines []line
	for i, raw := range strings.Split(src, "\n") {
		raw = strings.TrimRight(raw, "\r")
		trimmed := strings.TrimLeft(raw, " ")
		if strings.HasPrefix(trimmed, "\t") {
			return nil, errAt(i+1, "tab indentation is not supported")
		}
		t := strings.TrimRight(trimmed, " ")
		if t == "" || strings.HasPrefix(t, "#") || t == "---" {
			continue
		}
		lines = append(lines, line{indent: len(raw) - len(trimmed), text: t, no: i + 1})
	}
	if len(lines) == 0 {
		return nil, errors.New("empty YAML document")
	}
	p := &parser{lines: lines}
	v, err := p.block(lines[0].indent)
	if err != nil {
		return nil, err
	}
	if p.pos != len(p.lines) {
		return nil, errAt(p.lines[p.pos].no, "unexpected indentation")
	}
	return v, nil
}

func isSeqItem(t string) bool { return t == "-" || strings.HasPrefix(t, "- ") }

func (p *parser) block(indent int) (any, error) {
	if isSeqItem(p.lines[p.pos].text) {
		return p.seq(indent)
	}
	return p.mapping(indent)
}

func (p *parser) mapping(indent int) (any, error) {
	m := &Map{vals: map[string]any{}}
	for p.pos < len(p.lines) {
		l := p.lines[p.pos]
		if l.indent < indent || isSeqItem(l.text) {
			break
		}
		if l.indent > indent {
			return nil, errAt(l.no, "unexpected indentation")
		}
		key, rest, ok := splitKey(l.text)
		if !ok {
			return nil, errAt(l.no, "expected 'key: value'")
		}
		if _, dup := m.vals[key]; dup {
			return nil, errAt(l.no, "duplicate key %q", key)
		}
		p.pos++
		var val any
		var err error
		if rest == "" {
			val, err = p.child(indent)
		} else {
			val, err = parseScalar(rest, l.no)
		}
		if err != nil {
			return nil, err
		}
		m.keys = append(m.keys, key)
		m.vals[key] = val
	}
	return m, nil
}

// child parses the value that follows a "key:" line with nothing after it.
func (p *parser) child(parentIndent int) (any, error) {
	if p.pos < len(p.lines) {
		n := p.lines[p.pos]
		if n.indent > parentIndent {
			return p.block(n.indent)
		}
		if n.indent == parentIndent && isSeqItem(n.text) {
			return p.seq(parentIndent)
		}
	}
	return Scalar{Kind: KindNull}, nil
}

func (p *parser) seq(indent int) (any, error) {
	out := []any{}
	for p.pos < len(p.lines) {
		l := p.lines[p.pos]
		if l.indent < indent || !isSeqItem(l.text) {
			break
		}
		if l.indent > indent {
			return nil, errAt(l.no, "unexpected indentation")
		}
		rest := strings.TrimLeft(strings.TrimPrefix(l.text, "-"), " ")
		offset := len(l.text) - len(rest)
		var v any
		var err error
		switch {
		case rest == "":
			p.pos++
			if p.pos < len(p.lines) && p.lines[p.pos].indent > indent {
				v, err = p.block(p.lines[p.pos].indent)
			} else {
				v = Scalar{Kind: KindNull}
			}
		case isSeqItem(rest):
			return nil, errAt(l.no, "nested inline sequences are not supported")
		default:
			if _, _, ok := splitKey(rest); ok {
				// "- key: value" starts a mapping whose keys align with `key`.
				p.lines[p.pos] = line{indent: indent + offset, text: rest, no: l.no}
				v, err = p.mapping(indent + offset)
			} else {
				p.pos++
				v, err = parseScalar(rest, l.no)
			}
		}
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func splitKey(t string) (key, rest string, ok bool) {
	if t[0] == '"' || t[0] == '\'' {
		s, n, err := scanQuoted(t)
		if err != nil {
			return "", "", false
		}
		after := t[n:]
		if after == ":" {
			return s, "", true
		}
		if strings.HasPrefix(after, ": ") {
			return s, strings.TrimSpace(after[2:]), true
		}
		return "", "", false
	}
	if i := strings.Index(t, ": "); i >= 0 {
		return strings.TrimSpace(t[:i]), strings.TrimSpace(t[i+2:]), true
	}
	if strings.HasSuffix(t, ":") {
		return strings.TrimSpace(t[:len(t)-1]), "", true
	}
	return "", "", false
}

// scanQuoted decodes a quoted scalar at the start of t and returns the decoded
// text and the number of bytes consumed.
func scanQuoted(t string) (string, int, error) {
	q := t[0]
	if q == '\'' {
		var b strings.Builder
		for i := 1; i < len(t); i++ {
			if t[i] == '\'' {
				if i+1 < len(t) && t[i+1] == '\'' {
					b.WriteByte('\'')
					i++
					continue
				}
				return b.String(), i + 1, nil
			}
			b.WriteByte(t[i])
		}
		return "", 0, errors.New("unterminated single-quoted scalar")
	}
	for i := 1; i < len(t); i++ {
		switch t[i] {
		case '\\':
			i++
		case '"':
			s, err := strconv.Unquote(t[:i+1])
			if err != nil {
				return "", 0, fmt.Errorf("invalid double-quoted scalar: %v", err)
			}
			return s, i + 1, nil
		}
	}
	return "", 0, errors.New("unterminated double-quoted scalar")
}

func parseScalar(rest string, no int) (Scalar, error) {
	if rest[0] == '"' || rest[0] == '\'' {
		s, n, err := scanQuoted(rest)
		if err != nil {
			return Scalar{}, errAt(no, "%v", err)
		}
		if tail := strings.TrimSpace(rest[n:]); tail != "" && !strings.HasPrefix(tail, "#") {
			return Scalar{}, errAt(no, "unexpected text after quoted scalar")
		}
		return Scalar{Text: s, Kind: KindString}, nil
	}
	if i := strings.Index(rest, " #"); i >= 0 {
		rest = strings.TrimSpace(rest[:i])
	}
	if strings.ContainsRune("{[|>&*!%@`", rune(rest[0])) {
		return Scalar{}, errAt(no, "unsupported YAML construct starting with %q", rest[0])
	}
	return Scalar{Text: rest, Kind: classify(rest)}, nil
}

func classify(s string) Kind {
	switch s {
	case "", "~", "null", "Null", "NULL":
		return KindNull
	case "true", "True", "TRUE", "false", "False", "FALSE":
		return KindBool
	}
	if reInt.MatchString(s) {
		return KindInt
	}
	if reFloat.MatchString(s) {
		return KindFloat
	}
	return KindString
}
