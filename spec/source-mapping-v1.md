# Source mapping contract v1 (experimental)

This document defines the language-neutral behavior for source-mapping profiles
and interop vectors. It is normative for this experimental v1 contract. It does
not define a wire format for profiles or a protocol adapter.

## 1. Profile identity and units

A profile binds one exact source identity tuple:

- `protocol`: protocol/adapter identifier;
- `device`: configured device identity within that adapter's namespace;
- `point`: protocol-specific point address or node identifier.

The tuple is matched exactly. No case folding, alias expansion, address discovery,
or wildcard matching is implied. One profile per exact tuple is allowed in a
profile document.

`source_unit` labels the value supplied by the adapter before the configured
transform. `canonical_unit` must exactly equal the selected descriptor's unit.
Unit conversion is never inferred from labels. The adapter must transform any
protocol-specific encoding (for example SunSpec scale factors) before producing
a `SourceSample`, and the sample's `unit` must match `source_unit` exactly.

## 2. Numeric transform

Profile scale and offset are exact rational numbers:

- `scale = scale_numerator / scale_denominator`
- `offset = offset_numerator / offset_denominator`
- both denominators must be positive;
- scale numerator must be nonzero (negative scales are allowed);
- offset may be positive, negative, or zero.

The physical canonical value is computed exactly as:

```text
physical = raw_value * scale + offset
```

No binary floating-point arithmetic is permitted. `raw_value` is a plain
base-10 decimal string: optional leading sign, one or more ASCII digits, and
optionally a decimal point followed by one or more ASCII digits. Exponent
notation, whitespace, separators, a trailing decimal point, and more than 18
fractional digits are invalid. A leading plus sign is permitted.

The resulting physical value is encoded using the selected descriptor's existing
scale and offset. The encoded integer is rounded to nearest, with exact halfway
cases rounded away from zero. If the result is outside the descriptor's allowed
encoded range or int32, mapping fails; it must not clamp or wrap the value.

## 3. Source sample validation order

Implementations must reject a sample in this order:

1. source identity differs from the selected profile:
   `source_identity_mismatch`;
2. observed sample unit differs from the profile's `source_unit`:
   `observed_source_unit_mismatch`;
3. malformed decimal:
   `invalid_decimal`;
4. encoded result is outside the supported range:
   `out_of_range`.

The error strings in this list are the vector-format identifiers. An implementation
may expose native error types internally, but its vector runner must translate
them to these identifiers. Other profile/document validation errors are not
sample-vector outcomes and must fail profile loading.

## 4. Atom output

Successful mappings produce the existing six-byte OMWEI atom:

- bytes 0..1: descriptor ID, unsigned 16-bit big-endian;
- bytes 2..5: encoded value, signed 32-bit two's-complement big-endian.

The vector's `expected_wire_hex` is exactly 12 hexadecimal characters (six
bytes). Implementations compare decoded bytes, not letter casing in the hex
string. The mapping contract does not change the atom format.

## 5. Interop vector document

The top-level document contains `schema_version: "1"`, a non-empty descriptive
string, and a non-empty `cases` array. Each case has a unique non-empty `name`,
a `sample`, and exactly one of:

- `expected_wire_hex` for success; or
- `expected_error` with one of the identifiers in section 3.

A successful case may also include `expected_rounded` (boolean). When present,
it asserts whether descriptor encoding discarded a non-zero fractional
remainder before rounding. `true` means rounding was required, even if the
encoded integer is unchanged; `false` means the transformed value was exactly
representable in the descriptor's encoded resolution. Error cases do not have
a rounding outcome.

Unknown fields, duplicate case names, malformed JSON, multiple top-level JSON
values, missing/ambiguous outcomes, and unsupported error identifiers must fail
validation. JSON Schema validates document shape; the runner additionally checks
cross-field rules and actual mapping behavior.

## 6. Conformance and limits

A conforming implementation must run the checked-in vectors without changing
their expected outcomes. It should add its own negative cases for its parser,
numeric boundary, and protocol adapter. A passing vector suite proves only the
covered mapping contract. It does not prove correct device identity provisioning,
register addressing, coherent reads, freshness, transport security, or vendor
semantics.

The checked-in Modbus-style and SunSpec-style examples are synthetic. In
particular, the SunSpec adapter must resolve `W_SF` from the same coherent
model/device snapshot before creating the sample; the vector does not perform
SunSpec register I/O.
