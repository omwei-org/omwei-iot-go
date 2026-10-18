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

Keep in sync with the Rust crate: `scripts/sync-from-rust.sh <path-to-omwei_iot>`.

## Scope

Ported: descriptors, atoms, 6-byte and 16-byte encodings, decimal conversion,
corpus validation. Not ported (Rust-only tooling): the `omwei` CLI, benchmarks,
the no_std firmware example, JSON/CBOR observation helpers.

## License

Apache-2.0, same as the Rust crate.
