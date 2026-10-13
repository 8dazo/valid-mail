FROM node:24.19.0-bookworm-slim AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
# Optional public CA bundle supports builds behind a TLS-verifying egress proxy.
RUN --mount=type=secret,id=ca-certificates \
    if [ -s /run/secrets/ca-certificates ]; then export NODE_EXTRA_CA_CERTS=/run/secrets/ca-certificates; fi; \
    npm ci --no-fund
COPY frontend/ ./
RUN npm run build

FROM golang:1.27.2-bookworm AS server
WORKDIR /app
COPY go.mod go.sum ./
RUN --mount=type=secret,id=ca-certificates \
    if [ -s /run/secrets/ca-certificates ]; then export SSL_CERT_FILE=/run/secrets/ca-certificates; fi; \
    go mod download && go mod verify
COPY . .
COPY --from=frontend /app/frontend/dist ./frontend/dist
RUN CGO_ENABLED=0 go build -mod=readonly -trimpath -ldflags="-s -w" -o /weme .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=server /weme /weme
ENV PORT=8080 ENABLE_SMTP=true ENABLE_PROXY_SMTP=false AUTO_UPDATE_DISPOSABLE=false
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/weme"]
