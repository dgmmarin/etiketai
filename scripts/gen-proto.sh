#!/usr/bin/env bash
set -euo pipefail

PROTO_DIR="$(cd "$(dirname "$0")/.." && pwd)/proto"
OUT_DIR="$(cd "$(dirname "$0")/.." && pwd)/gen"

echo "Generating proto code from $PROTO_DIR..."

for proto_file in "$PROTO_DIR"/**/**/*.proto; do
  pkg_dir=$(dirname "$proto_file")
  rel=$(realpath --relative-to="$PROTO_DIR" "$pkg_dir")

  protoc \
    --proto_path="$PROTO_DIR" \
    --go_out="$OUT_DIR" \
    --go_opt=paths=source_relative \
    --go-grpc_out="$OUT_DIR" \
    --go-grpc_opt=paths=source_relative \
    "$proto_file"

  echo "  ✓ $rel"
done

echo "Done. Generated code is in $OUT_DIR"
