#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

PATH="$(go env GOPATH)/bin:$PATH" \
  protoc --proto_path=. \
  --go_out=paths=source_relative:. \
  --go-grpc_out=paths=source_relative:. \
  internal/doctor/proto/doctor.proto \
  internal/appointment/proto/appointment.proto
