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

## SunSpec Model 103 Modbus adapter (experimental)

The Go package includes a transport interface that matches the register-read
methods exposed by `github.com/otfabric/modbus`'s `*modbus.Client`, without
forcing that transport dependency on all users. The integration layer can pass
its existing client directly to `DiscoverSunSpecModel103Adapter`.

Discovery starts at the configured first **model-header** register (not the
SunSpec signature/common block) and is bounded by `MaxModels`. The returned
adapter then reads Model 103's `W` and `W_SF` together in one two-register
request and can feed that acquisition straight to the EABC mapper:

```go
source := omwei.SourceIdentity{
    Protocol: "sunspec-modbus",
    Device:   provisionedDeviceID, // provisioned out of band
    Point:    "model:103:W;scale-factor:W_SF",
}
adapter, err := omwei.DiscoverSunSpecModel103Adapter(
    ctx, existingModbusClient, source, unitID, omwei.SunSpecInputRegister,
    omwei.SunSpecModelDiscoveryConfig{
        FirstHeaderAddress: firstModelHeaderAddress,
        MaxModels:           64,
    },
)
if err != nil {
    return err
}
result, err := omwei.ReadAndMapSunSpecModel103ActivePower(adapter, mapping)
if err != nil {
    return err
}
atom := result.Atom
_ = atom
```

The optional production-client contract check lives in a nested module
so the core library remains dependency-free. It pins
`github.com/otfabric/modbus v1.0.0` (Go 1.21-compatible) and verifies at
compile/test time that its public `*modbus.Client` implements
`SunSpecModel103RegisterClient`:

```sh
cd integration/otfabric-modbus
go mod tidy
go test ./... -count=1
```

The CI workflow runs this check independently of the root module. This is an
API compatibility check, not a live transport test against the upstream
client; the root module's localhost simulator currently tests our own
minimal test client.

A localhost Modbus TCP simulator test exercises the MBAP frame, function
code 4/3 register reads, transaction ID, unit ID, model discovery, and final
mapping without hardware. Run it with:

```sh
go test ./... -run 'TestSunSpecModel103OverLocalModbusTCPToEABCAtom|TestDiscoverSunSpecModel103|TestSunSpecModel103ModbusAdapter'
```

The TCP test suite also injects malformed responses: Modbus exception
frames, truncated exception frames, mismatched transaction IDs and Unit IDs,
unexpected function codes, incorrect byte counts, and invalid MBAP lengths.
The client must fail closed for every injected fault.

The simulator uses synthetic registers and is deliberately test-only; it is
not a production Modbus server and does not replace testing against a vendor
device or a standards-validated simulator.

Choose `SunSpecInputRegister` or `SunSpecHoldingRegister` according to the
device's documented register table. Discovery validates the model chain's
bounds and Model 103's minimum point length; it does not prove the device's
cryptographic identity or certify that a vendor's register layout matches the
standard. The snapshot ID labels one successful client-side acquisition; one
Modbus request does not prove the device updated `W` and `W_SF` atomically.
Tests use a fake register client, so passing them is not hardware-in-the-loop
validation.

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

The normative, language-neutral rules for decimal syntax, rational transforms,
rounding, validation order, and vector behavior are in
[`spec/source-mapping-v1.md`](spec/source-mapping-v1.md). Treat this API and
specification as experimental until the v1 contract has been reviewed and
implemented by an independent consumer.

## Run the interop vectors

Run the same checked-in source mapping vectors against the Go implementation:

```sh
go run ./cmd/source-mapping-vectors
```

The command exits non-zero on the first mismatch and prints each accepted
wire result or expected rejection. Alternate documents can be supplied with
`-profiles path/to/profiles.json` and `-vectors path/to/vectors.json`.
The vector format is language-neutral; this CLI is the Go runner, not itself
proof that another implementation conforms.

## Interop contract and conformance

The normative, language-neutral rules are in
[`spec/source-mapping-v1.md`](spec/source-mapping-v1.md). The profile and vector
formats are defined by the JSON Schemas under `schemas/`. The checked-in
`testdata/source-mapping-interop-v1.json` file is synthetic and covers successful
mapping plus identity, unit, decimal, and range rejection. Run the Go runner
with `go run ./cmd/source-mapping-vectors`; its tests also verify that malformed
vector documents and mismatched wire results fail. Passing this Go runner is
not proof of conformance by an independent implementation or of a real OT Fabric
adapter.


## Scope

Ported: descriptors, atoms, 6-byte and 16-byte encodings, decimal conversion,
corpus validation. Not ported (Rust-only tooling): the `omwei` CLI, benchmarks,
the no_std firmware example, JSON/CBOR observation helpers.

## License

Apache-2.0, same as the Rust crate.
