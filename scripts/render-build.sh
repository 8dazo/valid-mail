#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

if ! command -v node >/dev/null || ! command -v npm >/dev/null; then
  echo "Render build requires Node.js 24 and npm. Set NODE_VERSION=24.19.0 in the service environment." >&2
  exit 1
fi
node -e 'if (Number(process.versions.node.split(".")[0]) < 24) { console.error("Render build requires Node.js 24+. Set NODE_VERSION=24.19.0."); process.exit(1); }'

echo "Building WeMe frontend ($(node --version))"
npm ci --prefix frontend --no-fund
npm run build --prefix frontend
test -s frontend/dist/index.html

echo "Verifying Go dependencies and building the embedded server"
go mod download
go mod verify
mkdir -p bin
CGO_ENABLED=0 go build -buildvcs=false -mod=readonly -trimpath -ldflags="-s -w" -o bin/valid-mail .
