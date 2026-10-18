# Source mapping and interop vectors v1 (experimental)

This document defines the behaviour expected of implementations that claim
compatibility with the experimental OMWEI source-mapping v1 contract. It does
not change the six-byte OMWEI atom format or claim an OT Fabric integration.

## 1. Source identity and sample

A source identity is the exact tuple `(protocol, device, point)`. These are
opaque, case-sensitive strings in v1; implementations MUST NOT silently trim,
case-fold, alias, or infer identity. A sample MUST match the configured tuple
exactly before its value is mapped.

`SourceSample.value` is a plain base-10 decimal string. It permits an optional
leading `+` or `-`, digits before the decimal point, and optionally a decimal
point followed by digits. Exponents, whitespace, a missing whole-number part,
and a trailing decimal point are invalid. At most 18 fractional digits are
accepted. The current Go parser also bounds decimal magnitude to signed 128-bit
intermediate representation; interoperable implementations MUST reject values
that exceed the declared decimal input limits.

The sample's `unit` MUST exactly equal the configured `source_unit`. Unit
labels are identifiers, not a conversion engine: implementations MUST NOT
infer a conversion from a label. The configured rational transform performs
the conversion.

## 2. Mapping profile

Each profile contains:

- source identity: protocol, device, point;
- a known `descriptor_id`;
- `source_unit` and the descriptor's exact `canonical_unit`;
- non-zero `scale_numerator`, positive `scale_denominator`;
- `offset_numerator` and positive `offset_denominator`.

The configured transform is:

```text
physical = raw * scale_numerator / scale_denominator
         + offset_numerator / offset_denominator
```

All operations MUST use exact rational/decimal arithmetic until descriptor
encoding. Binary floating-point MUST NOT be used. The source adapter is
responsible for decoding protocol-specific representations (for example,
SunSpec integer points and their scale factors) and for supplying a coherent
sample. The generic mapper does not read devices or resolve registers.

A document MUST declare `schema_version: "1"`, contain at least one mapping,
reject unknown fields, and reject duplicate source identities. A profile's
canonical unit MUST equal the selected descriptor's canonical unit. Unknown
descriptors and invalid denominators MUST be rejected.

## 3. Descriptor encoding and atom

After the source transform, the canonical physical value is encoded according
to the selected descriptor:

```text
encoded = round_away_from_zero((physical - descriptor_offset)
                               * descriptor_scale)
```

The encoded integer MUST fit signed int32 and the descriptor's declared
minimum/maximum encoded range. An implementation MUST reject overflow and
out-of-range values; it MUST NOT clamp, wrap, saturate, or silently emit an
atom.

The v1 atom is six bytes: unsigned 16-bit descriptor ID followed by signed
32-bit encoded value, both big-endian. The published vectors compare all six
bytes. A halfway rounding case is rounded away from zero, for both positive
and negative values.

## 4. Interop vector document

A vector document MUST declare `schema_version: "1"` and at least one case.
Each case has a unique descriptive name and one sample, and exactly one of:

- `expected_wire_hex`: exactly 12 hexadecimal characters representing the
  complete six-byte atom; or
- `expected_error`: one of the stable error names below.

The current v1 error names are:

- `source_identity_mismatch`
- `observed_source_unit_mismatch`
- `invalid_decimal`
- `out_of_range`

A runner MUST fail non-zero if a case has an unexpected success, unexpected
error, incorrect wire bytes, invalid input document, or missing profile needed
to evaluate the case. A runner MUST NOT treat any error as equivalent to the
expected error.

## 5. SunSpec adapter boundary

For SunSpec scaled integer points, the adapter MUST resolve the point and its
scale factor from a coherent read/snapshot and apply the model's scale-factor
rule before creating `SourceSample`. The generic mapper receives the resulting
engineering value and its declared unit. A synthetic SunSpec vector proves
only the mapper contract after scale-factor resolution; it is not evidence of
device register addressing, coherent acquisition, or vendor conformance.

## 6. Status and non-goals

This v1 contract is experimental. It does not specify authentication, source
attestation, freshness, quality flags, timestamps, transport discovery,
authorization, command execution, or OT Fabric APIs. Those concerns MUST be
specified and tested separately at the integration boundary.
