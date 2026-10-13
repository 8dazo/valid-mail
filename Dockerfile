FROM golang:1.27.2-bookworm AS toolkit
WORKDIR /app
COPY go.mod go.sum ./
# Optional trust bundle for TLS-verifying cloud build proxies.
RUN --mount=type=secret,id=ca-certificates \
    if [ -s /run/secrets/ca-certificates ]; then export SSL_CERT_FILE=/run/secrets/ca-certificates; fi; \
    go mod download && go mod verify
COPY internal/toolkit ./internal/toolkit
COPY cmd/weme-wasm ./cmd/weme-wasm
COPY scripts/build-wasm.sh ./scripts/build-wasm.sh
RUN bash scripts/build-wasm.sh

FROM node:24.19.0-bookworm-slim AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN --mount=type=secret,id=ca-certificates \
    if [ -s /run/secrets/ca-certificates ]; then export NODE_EXTRA_CA_CERTS=/run/secrets/ca-certificates; fi; \
    npm ci --no-fund
COPY frontend/ ./
COPY --from=toolkit /app/frontend/public/wasm ./public/wasm
COPY --from=toolkit /app/frontend/src/generated ./src/generated
# The Go stage already built the verified, fingerprinted toolkit assets.
RUN npm run build --ignore-scripts

FROM toolkit AS server
COPY . .
COPY --from=frontend /app/frontend/dist ./frontend/dist
RUN CGO_ENABLED=0 go build -buildvcs=false -mod=readonly -trimpath -ldflags="-s -w" -o /weme .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=server /weme /weme
ENV PORT=8080 ENABLE_SMTP=true ENABLE_PROXY_SMTP=false AUTO_UPDATE_DISPOSABLE=false
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/weme"]
