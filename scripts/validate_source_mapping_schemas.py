#!/usr/bin/env python3
"""Validate the published JSON Schemas and their canonical example documents."""
import json
from pathlib import Path

from jsonschema import Draft202012Validator

ROOT = Path(__file__).resolve().parents[1]
CHECKS = (
    (
        "schemas/source-mappings-v1.schema.json",
        "examples/source-mappings-v1.json",
    ),
    (
        "schemas/source-mapping-interop-v1.schema.json",
        "testdata/source-mapping-interop-v1.json",
    ),
)


def load_json(path: str):
    with (ROOT / path).open(encoding="utf-8") as stream:
        return json.load(stream)


def main() -> None:
    for schema_path, document_path in CHECKS:
        schema = load_json(schema_path)
        Draft202012Validator.check_schema(schema)
        document = load_json(document_path)
        errors = sorted(
            Draft202012Validator(schema).iter_errors(document),
            key=lambda error: list(error.absolute_path),
        )
        if errors:
            details = "\n".join(
                f"  {document_path}:{list(error.absolute_path)}: {error.message}"
                for error in errors
            )
            raise SystemExit(f"Schema validation failed:\n{details}")
        print(f"OK: {document_path} conforms to {schema_path}")


if __name__ == "__main__":
    main()
