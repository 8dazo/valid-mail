#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."
mkdir -p bin
go build -buildvcs=false -mod=readonly -trimpath -o bin/weme-toolkit ./cmd/weme-toolkit
