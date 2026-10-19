# Mapping Contract v2 — Draft Proposal

**Status:** Draft / Proposal. Not normative and not implemented.  
**Scope:** Documentation only. This file does not change the shared corpus, generated registry, atom wire format, Go implementation, or Rust implementation.

## 1. Verified current corpus and QUDT references

Before proposing changes, the current `registry/corpus-v0.1.yaml` was inspected. Its existing `semantic.mappings` entries are quantity-kind mappings, and currently reference these QUDT concepts:

| Descriptor | Current `quantity_kind` | Existing QUDT concept |
|---|---|---|
| `0x0042` `ambient_temperature` | `temperature` | `Temperature` |
| `0x0043` `relative_humidity` | `relative_humidity` | `RelativeHumidity` |
| `0x0044` `atmospheric_pressure` | `pressure` | `Pressure` |
| `0x0045` `active_power` | `power` | `Power` |
| `0x0046` `vibration_rms` | `vibration_velocity` | `Velocity` |
| `0x0047` `methane_concentration` | `mole_fraction` | `MoleFraction` |

**Verified fact:** the corpus already maps `quantity_kind: power` to QUDT `Power` for descriptor `0x0045`. Do not describe generic power as lacking a QUDT mapping, and do not add a duplicate mapping merely to repeat it.

The inspected entries do **not** currently define QUDT mappings for canonical units; their QUDT references are under semantic quantity-kind mappings. Any future unit-vocabulary alignment is a separate question and must be justified against the corpus schema and the actual QUDT terms, not assumed from the presence of a quantity-kind mapping.

## 2. Proposed signed, finer-resolution active-power descriptor

The current descriptor `0x0045` is active, uses canonical unit `watt`, scale `1`, and has the allowed encoded range `0..100,000,000`. It therefore does not represent negative active power and does not offer tenths-of-a-watt resolution.

**Proposal only:** introduce a new versioned descriptor rather than silently changing the meaning or compatibility of `0x0045`. Its ID and version must be allocated during a coordinated corpus change; no ID is reserved by this document.

Candidate encoding:

| Field | Proposal |
|---|---|
| Label | `active_power_signed` (candidate) |
| Quantity kind | `active_power` (candidate; see semantic note below) |
| Canonical unit | `watt` |
| Encoding datatype | signed `int32` |
| Encoding scale | `0.1` W per encoded integer |
| Physical engineering range | `-100,000,000 W .. +100,000,000 W` |
| Corresponding encoded integer range | `-1,000,000,000 .. +1,000,000,000` |

The encoded limits fit within signed `int32`. The corpus uses the descriptor's encoding scale and integer constraints; the engineering range and encoded range must both be reviewed explicitly before implementation. These are proposed values, not current corpus facts.

### Sign convention (must be explicit)

**SunSpec Model 103 convention verified:** the `W` point is signed active power scaled by `W_SF`. SunSpec's DER sign convention defines positive active power as generation/injection by the DER and negative active power as absorption by the DER. See the [SunSpec DER Information Model Specification](https://sunspec.org/wp-content/uploads/2021/02/SunSpec-DER-Information-Model-Specification-V1-0-02-01-2021.pdf), which states that injection is positive and absorption is negative.

For this proposal, positive means active power flowing **out of the declared DER/device boundary** and negative means active power flowing **into it**. These descriptions align only when the measured boundary is the DER's AC interface and the source follows the SunSpec generator-reference convention. Do not generalize this sign convention to arbitrary devices or protocols without checking their documented reference direction.

The source mapping must identify the device and relevant measurement boundary. An adapter may invert sign only when the source's documented convention requires it and that transformation is explicit in the mapping. Never infer sign from a point label, device class, or observed values. Where a device exposes multiple phases or electrical boundaries, aggregation and boundary selection must be documented separately; this descriptor alone does not define them.

## 3. Keep active, apparent, and reactive power distinct

Do not collapse these measurements into one undifferentiated power kind:

- **Active power** — real/active power, expressed in watts (W).
- **Apparent power** — apparent power, expressed in volt-amperes (VA).
- **Reactive power** — reactive power, expressed in vars (var).

The quantity-kind distinction is semantic; the unit distinction is dimensional. A mapping must not convert VA or var into W merely because all three are informally called “power”. Proposed labels and quantity-kind names are not approved corpus vocabulary.

The currently inspected corpus has only the generic `power` quantity kind and QUDT `Power` for `0x0045`. Before adopting more specific quantity kinds or QUDT concepts for active/apparent/reactive power, verify the exact terms against the chosen QUDT release and document the mapping rationale. Do not claim that those more specific concepts are already present in the corpus.

## 4. Existing rounding contract

Rounding is already observable through `ConversionResult.Rounded`; v2 should preserve that behavior rather than inventing a second rounding flag.

- Exact conversion to the descriptor's integer resolution: `Rounded == false`.
- Any non-exact conversion that is rounded to the representable integer: `Rounded == true`.
- Midpoint behavior must match the existing Go/Rust shared contract (currently tested as half away from zero); parity tests must cover positive and negative ties.
- Range checking occurs against the descriptor's encoded constraints after conversion. A value outside the permitted range must fail; it must not wrap or clamp.

The finer resolution changes the encoding scale and permissible integer range, not the atom wire layout. The existing atom format remains unchanged unless a separately reviewed, cross-language proposal explicitly establishes otherwise.

## 5. Rust parity plan (proposal; no implementation in this step)

The corpus is shared with the Rust crate `Equinibrium/omwei_iot`; Go generates its registry from the same YAML and runs shared JSON vectors. A corpus change must not be merged in only one language.

### Sequence and acceptance gates

1. **Agree the semantics first.** Review the proposed signed range, 0.1 W resolution, sign convention, quantity-kind names, units, and compatibility strategy. Confirm QUDT terms and schema representation.
2. **Allocate and update the shared corpus only after approval.** The corpus/semantic owner must explicitly allocate the new descriptor ID in the coordinated corpus change; this document reserves no ID. Keep `0x0045` unchanged. Publish the same corpus content to both repositories and record its version and commit/hash in each. Do not edit the corpus as part of this documentation-only proposal.
3. **Update shared vectors in both repositories in the same change sequence.** Add positive, negative, zero, minimum/maximum, just-outside-range, exact-resolution, non-exact rounding, positive/negative midpoint, invalid decimal, and sign-inversion cases. Include expected canonical integer values, `Rounded`, error outcome, and encoded atom bytes where applicable. Go and Rust must consume equivalent logical vectors, with protocol-specific fixtures where needed.
4. **Update the Rust crate and vectors.** Update its corpus-driven generation/build path and conformance tests; run the Rust test suite and vectors against the approved corpus hash.
5. **Update the Go core and vectors.** Regenerate the registry, run corpus/schema validation, Go tests, interop vectors, `CGO_ENABLED=0 go build ./...`, and the Rust differential suite against that same corpus hash.
6. **Run cross-language parity.** Execute the equivalent vectors in both implementations and compare outcomes and atom bytes. Include the existing differential harness; record exact refs and logs.
7. **Review and release together.** Both implementations, their shared vectors, and the agreed corpus version must pass before either side claims v2 support.

### Owners / responsibilities

- **Corpus/semantic owner (descriptor-ID allocator):** approves descriptor identity, ranges, quantity-kind distinctions, sign convention, QUDT references, corpus version, and allocates the new descriptor ID in the reviewed shared-corpus change.
- **Rust maintainer:** integrates the approved corpus and equivalent shared vectors into Rust generation/build and tests.
- **Go maintainer:** integrates the same corpus and equivalent vectors into Go generated registry and tests.
- **Parity reviewer:** checks byte-for-byte atom equality, `Rounded` behavior, error parity, matching corpus hashes, and recorded test evidence.

These are proposed role assignments, not claims that changes have been implemented or approved.

## 6. Non-goals and limitations

This proposal does not change the corpus, assign a descriptor ID, change the six-byte atom format, implement new quantity kinds, assert vendor sign conventions, add QUDT unit mappings, or claim Rust/Go v2 parity. The current descriptor `0x0045` remains unchanged until a separately reviewed implementation proposal is approved.
