#!/usr/bin/env bash
# Pull the shared source-of-truth files from a checkout of the Rust crate and
# regenerate. The YAML corpus and the JSON vectors are the contract between the
# two implementations; nothing else is copied.
set -euo pipefail
RUST_REPO=${1:?path to omwei_iot checkout}
cp "$RUST_REPO/registry/corpus-v0.1.yaml" registry/
cp "$RUST_REPO"/tests/vectors/*.json testdata/
go generate ./...
go test ./...
