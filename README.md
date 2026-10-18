# omwei-iot-go

Go port of the Rust crate [`omwei-iot`](https://github.com/Equinibrium/omwei_iot):
semantic interoperability primitives for heterogeneous IoT systems.

- **Pure Go, standard library only, no cgo.** Builds with `CGO_ENABLED=0`.
  The runtime package does not parse YAML.
- **One source of truth.** `registry/corpus-v0.1.yaml` (shared with the Rust
  crate) is compiled into `registry_generated.go` by `cmd/omwei-gen`, the Go
  counterpart of the Rust `build.rs`.
- **Same contract.** The Rust crate's JSON conformance vectors
  (`testdata/*.json`) run unchanged against this port.

```go
import omwei "github.com/omwei-org/omwei-iot-go"

// 22.5004 °C -> encoded integer 22500 (0.001 °C steps), exact base-10 arithmetic.
r, err := omwei.FromDecimalWithReport(omwei.AmbientTemperatureID, "22.5004")
// r.Atom.CanonicalValue == 22500, r.Rounded == true

wire := omwei.Encode(r.Atom)       // [6]byte: uint16 ID | int32, big-endian
back, err := omwei.Decode(wire[:]) // validates ID and range
```

Errors are sentinels for `errors.Is`: `ErrUnknownDescriptor`, `ErrOutOfRange`,
`ErrInvalidAtomLength`, `ErrInvalidDecimal`.

## How the "automatic conversion" works

General Rust-to-Go source translation is not reliable, so the port is split by
what actually has to be identical:

| Concern | Mechanism |
|---|---|
| Descriptor table | Generated from the shared YAML (`go generate`) |
| Corpus validation | `corpus` package, ported rule for rule, same error texts |
| Atom wire format, decimal rounding | Small hand port, exact `math/big` arithmetic |
| Behavioural parity | Shared JSON vectors plus a differential test against the Rust code |
| Drift | `TestGeneratedTableMatchesCorpus` fails if table and YAML diverge |

Differential test: 200,000 random cases (valid and malformed decimals, boundary
values, ties, 40+ digit inputs, atom construction, atom decoding) give
byte-identical results in Rust and Go. Reproduce with `scripts/difftest.sh <rust-ref-binary>`;
`scripts/gen_cases.py` generates the cases deterministically (fixed seed).

CI (`.github/workflows/ci.yml`) builds the real Rust crate at a pinned commit
(tag `v0.1.2`) and runs the same differential test on every push. To run it
locally: `git clone https://github.com/Equinibrium/omwei_iot rust-src && git -C rust-src checkout 446ebe4`,
then `cargo build --release --manifest-path rustref/Cargo.toml` and
`scripts/difftest.sh rustref/target/release/omwei-rustref`.

Keep in sync with the Rust crate: `scripts/sync-from-rust.sh <path-to-omwei_iot>`.

## Source mapping profiles (experimental)

The source-mapping API separates upstream point identity and raw-value transforms
from the canonical OMWEI descriptor. An illustrative, versioned JSON profile is
available at `examples/source-mappings-v1.json`. It demonstrates two distinct
source points mapping to descriptor `0x0045` (`active_power`); it is not a verified vendor
register map.

Load and validate a profile with:

```go
import (
    "os"

    omwei "github.com/omwei-org/omwei-iot-go"
)

func mapExample() error {
    data, err := os.ReadFile("examples/source-mappings-v1.json")
    if err != nil {
        return err
    }
    doc, err := omwei.ParseSourceMappingDocument(data)
    if err != nil {
        return err
    }
    result, err := omwei.MapSourceValue(doc.Mappings[0], "1230")
    if err != nil {
        return err
    }
    wire := omwei.Encode(result.Atom) // existing six-byte atom format
    _ = wire
    return nil
}
```

The transform is explicitly configured as
`canonical physical value = raw value * scale + offset`; the descriptor then
applies its own encoding scale/offset and range validation. The parser rejects
unknown JSON fields, unsupported schema versions, invalid profiles, duplicate
source identities, and canonical-unit mismatches. It does not infer unit
conversions from unit labels, verify vendor-specific point semantics, or replace
device documentation. Treat profiles as configuration that requires review.

## SunSpec scale factors

The SunSpec model definition used for this example is the public
[SunSpec Models repository, Model 103 JSON](https://github.com/SunSpec/models/blob/master/json/model_103.json),
which defines the `W` AC power point in watts and associates it with `W_SF`.
This confirms the point/scale-factor relationship only; it does not certify any
specific device's register base address, firmware behavior, or live values.

SunSpec integer points can use a separate scale-factor point (for example,
`W` together with `W_SF`). Resolve the scale factor from the same model/device
snapshot and apply `engineering value = raw × 10^scale factor` before passing
the decimal string to `MapSourceValue`. The helper
`DecodeSunSpecScaledInteger` performs this step without binary floating point.
The example profile identifies the model point and scale-factor dependency, but
the caller remains responsible for reading a coherent pair and validating the
model/point layout against the applicable SunSpec model specification.

## Transport-neutral adapter contract

Protocol adapters should normalize protocol-specific payloads into a
`SourceSample` containing the exact configured `SourceIdentity`, a plain
decimal `Value`, and the `Unit` of that value. Resolve protocol-specific
scale factors (such as SunSpec `W_SF`) before creating the sample. Then call
`MapSourceSample(mapping, sample)`; it rejects source-identity and observed-unit
mismatches before mapping and constructing the canonical atom.

This boundary intentionally does not perform network I/O, discover points,
authenticate devices, establish data freshness, or infer unit conversions.
Those responsibilities remain with the adapter/integration layer and must be
covered by protocol-specific conformance tests.

## Transport-neutral interop contract

`SourceSample` is the adapter boundary: it carries the configured source
identity, a decimal-string value, and the unit of that value. The adapter must
resolve protocol-specific encodings (including SunSpec `W_SF`) before
constructing the sample. `MapSourceSample` rejects identity and observed-unit
mismatches before applying the configured rational transform.

The shared test vectors in `testdata/source-mapping-interop-v1.json` cover
successful Modbus-style and SunSpec-style samples that converge on the same
six-byte atom, as well as negative identity and unit checks. JSON Schemas for
profiles and vectors live under `schemas/` so non-Go implementations can validate
the same documents. Run them with
`go test ./... -count=1`. These are synthetic contract vectors, not a captured
device session or proof of vendor register addressing. A future OT Fabric
adapter should run the same vectors at its integration boundary.

## Scope

Ported: descriptors, atoms, 6-byte and 16-byte encodings, decimal conversion,
corpus validation. Not ported (Rust-only tooling): the `omwei` CLI, benchmarks,
the no_std firmware example, JSON/CBOR observation helpers.

## License

Apache-2.0, same as the Rust crate.
