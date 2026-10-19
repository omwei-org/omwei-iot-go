# Cross-Protocol Mapping Plan — Draft Proposal

**Status:** Draft / Plan only.  
**Scope:** Documentation only. No adapter, OPC UA implementation, corpus, vector, registry, or atom-wire changes are made here.

## 1. Goal

Demonstrate that the same physical measurement, acquired through different protocol adapters, can be normalized to the same canonical OMWEI descriptor and produce the same atom when the semantics and value are equivalent.

Initial comparison:

- **SunSpec:** companion adapter using `github.com/otfabric/go-sunspec`; resolve the Model 103 active-power point `W` and scale factor `W_SF`.
- **OPC UA:** a future adapter that reads an explicitly configured node and normalizes its value, unit, sign convention, and identity.
- **Shared target:** the same canonical descriptor and existing source-mapping contract in `omwei-iot-go`.

This is a plan, not a claim that an OPC UA adapter or cross-protocol integration exists.

## 2. Shared normalization boundary

Each protocol adapter should produce a transport-neutral `SourceSample`:

- `SourceIdentity`: protocol name, provisioned device identity, and configured point identity;
- exact decimal `Value`, after protocol-specific scale factor and any documented sign transform;
- observed `Unit` for that value.

Then pass the sample through `MapSourceSample` with a reviewed `SourceMapping`. The mapper validates identity and observed unit and applies the canonical descriptor's encoding, range, and rounding contract.

Suggested protocol identities:

- SunSpec: protocol `sunspec`; point identifies Model 103 `W` and its dependency on `W_SF`.
- OPC UA: protocol `opcua`; point identifies a configured namespace URI and node identifier, not a display name alone.

The exact identity strings are configuration examples, not a new wire contract. Device identity must be provisioned and must not be inferred solely from a network endpoint or node label.

## 3. Equivalence test design

Use a shared, versioned vector set once the cross-protocol proposal is approved. Each logical case should specify:

- logical measurement and device-relative boundary;
- protocol-specific raw input and scale/unit metadata;
- expected normalized exact decimal value;
- expected canonical descriptor ID and integer value;
- expected `Rounded` flag and expected error (if any);
- expected encoded atom bytes for accepted cases.

For a basic active-power case, both paths should normalize to the same unit (W), sign convention, and exact decimal value before mapping. For example, the SunSpec path can use a synthetic `W=1234`, `W_SF=-1` acquisition, yielding `123.4 W`. The OPC UA fixture should supply an equivalent documented value of `123.4 W`. The comparison must use a descriptor that actually permits the test value and its resolution; do not silently assume that the current non-negative, 1 W descriptor `0x0045` can represent negative values or tenths of a watt.

Compare the final atom bytes, not only display strings or floating-point values. A shared logical vector may have protocol-specific fixture data, but its expected canonical result must be identical.

## 4. Required cases

1. Equivalent exact values from SunSpec and OPC UA produce identical atoms.
2. SunSpec scale-factor decoding uses exact decimal arithmetic; no `float64` value path.
3. OPC UA values are normalized from the node's declared/validated datatype and engineering-unit metadata; no assumptions from a display label alone.
4. Equivalent sign conventions produce equal results; a documented source convention requiring inversion is tested explicitly.
5. Unit mismatch, missing scale factor, missing/ambiguous OPC UA unit metadata, invalid raw type, unimplemented SunSpec point, and out-of-range values fail closed.
6. Rounding and positive/negative midpoint behavior match the shared Go/Rust contract where applicable.
7. Invalid, stale, or untrusted acquisition context is rejected by the appropriate acquisition/policy layer; do not imply that `SourceSample` by itself authenticates a device or proves freshness.
8. Repeat the shared canonical vectors against Go and Rust to detect language drift.

## 5. Implementation sequence and owners

1. **Contract owner:** approve semantic identity, device-relative sign convention, units, quantity kind, and descriptor compatibility before adding fixtures.
2. **SunSpec adapter owner:** document the precise source point and `W_SF` dependency; use one coherent acquisition snapshot and preserve the existing limitations around device-side atomicity and freshness.
3. **OPC UA adapter owner:** define node identity, datatype validation, engineering-unit extraction, value conversion, and failure behavior. Select an OPC UA Go library only in the implementation proposal; this document introduces no dependency.
4. **Vector owner:** create shared logical cases and protocol-specific fixture inputs after the contract is approved; keep corpus and wire format unchanged until the corresponding cross-language change is reviewed.
5. **Go/Rust maintainers:** run the same canonical vectors in both implementations and compare atom bytes, `Rounded`, and errors.
6. **Integration reviewer:** verify both protocols map to the same descriptor under equivalent semantics; record exact repository refs, test commands, and logs.
7. **Release owner:** report synthetic-test, simulator, hardware-in-loop, and vendor-validation status separately. Do not describe one level of evidence as another.

## 6. Evidence boundaries and non-goals

- The existing SunSpec tests are synthetic and the Modbus integration harness uses a localhost simulator; they do not establish hardware-in-the-loop behavior or vendor certification.
- No OPC UA adapter or cross-protocol parity implementation is delivered by this plan.
- No claims are made about device authentication, source freshness, atomic measurement updates, or sign convention without separately implemented and tested evidence.
- Do not modify `registry/corpus-v0.1.yaml`, generated registries, shared vectors, or the atom wire format as part of this documentation-only step.
