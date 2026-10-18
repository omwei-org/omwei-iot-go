# Compatibility evidence: otfabric/go-modbus v1.1.3

**Record type:** integration / CI evidence  
**Recorded:** 2026-10-18  
**Repository:** `omwei-org/omwei-iot-go`  
**Status:** CI-verified for the commit and test scope listed below; not hardware-validated.

## Change under test

- Main merge commit: [`c7609d8b463493dfaaa010762ef6072c5d2a2043`](https://github.com/omwei-org/omwei-iot-go/commit/c7609d8b463493dfaaa010762ef6072c5d2a2043)
- Isolated integration module: [`integration/otfabric-modbus/go.mod`](../integration/otfabric-modbus/go.mod)
- Dependency: `github.com/otfabric/go-modbus v1.1.3`
- The integration module declares Go `1.23`; this keeps the dependency's toolchain requirement scoped to the integration harness rather than changing the root module's Go baseline.
- Compatibility contract: [`adapter_contract_test.go`](../integration/otfabric-modbus/adapter_contract_test.go)
- Client integration test: [`production_client_e2e_test.go`](../integration/otfabric-modbus/production_client_e2e_test.go)

## CI evidence

The GitHub Actions workflow run for the merge commit is [run 39139994100](https://github.com/omwei-org/omwei-iot-go/actions/runs/39139994100).

- Event: push to `main`
- Head SHA: `c7609d8b463493dfaaa010762ef6072c5d2a2043`
- Workflow conclusion: **success**
- Run timestamp: 2026-10-18 15:19:56 UTC; completed 15:20:33 UTC

The recorded workflow run is successful. This is evidence that the configured CI jobs passed for this commit; it is not a claim that every possible behavior was tested.

## Test scope represented by the integration harness

The integration harness uses the upstream OT Fabric Modbus client against a local Modbus TCP simulator. The covered scenario includes SunSpec Model 103 decoding and scaled-value handling (raw `W = 1234`, `W_SF = -1` gives `123 W`), along with protocol/error-path and client-contract checks implemented by the linked tests.

The simulator is not a physical inverter. Results establish compatibility only for the test cases and software revision identified here.

## Limits and non-claims

This record does **not** establish:

- operation against a physical inverter or any particular vendor/device firmware;
- successful commissioning or behavior in a production deployment;
- exhaustive Modbus/SunSpec conformance;
- security assurance, absence of vulnerabilities, or correctness outside the exercised paths;
- independent third-party audit or certification.

A future hardware-in-the-loop run should record the device model and firmware, connection settings, test inputs/outputs, test environment, and logs as separate evidence.
