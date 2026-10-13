#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."
output=frontend/public/wasm
mkdir -p "$output" frontend/src/generated
scratch=$(mktemp -d)
trap 'rm -rf "$scratch"' EXIT
GOOS=js GOARCH=wasm CGO_ENABLED=0 go build -buildvcs=false -mod=readonly -trimpath -ldflags="-s -w" -o "$scratch/toolkit.wasm" ./cmd/weme-wasm
gowasm_runtime="$(go env GOROOT)/lib/wasm/wasm_exec.js"
if [[ ! -f "$gowasm_runtime" ]]; then gowasm_runtime="$(go env GOROOT)/misc/wasm/wasm_exec.js"; fi
cp "$gowasm_runtime" "$scratch/go-runtime.js"
wasm_hash=$(sha256sum "$scratch/toolkit.wasm" | cut -c1-16)
js_hash=$(sha256sum "$scratch/go-runtime.js" | cut -c1-16)
# Only remove this generator's old, fingerprinted build artifacts.
find "$output" -maxdepth 1 -type f -regextype posix-extended -regex '.*/(toolkit-[a-f0-9]{16}\.wasm(\.gz)?|go-runtime-[a-f0-9]{16}\.js)' -delete
cp "$scratch/toolkit.wasm" "$output/toolkit-$wasm_hash.wasm"
gzip -n -9 -c "$scratch/toolkit.wasm" > "$output/toolkit-$wasm_hash.wasm.gz"
cp "$scratch/go-runtime.js" "$output/go-runtime-$js_hash.js"
cat > frontend/src/generated/go-runtime.json <<MANIFEST
{
  "wasm": "/wasm/toolkit-$wasm_hash.wasm",
  "runtime": "/wasm/go-runtime-$js_hash.js"
}
MANIFEST
printf 'Local Go toolkit built: %s bytes; assets fingerprinted.\n' "$(wc -c < "$scratch/toolkit.wasm")"
