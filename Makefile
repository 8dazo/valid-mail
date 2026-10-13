.PHONY: install frontend build test dev

install:
	cd frontend && npm ci
	go mod download

frontend:
	cd frontend && npm run build

build: frontend
	go build -mod=readonly -trimpath -o bin/weme .

test: frontend
	cd frontend && npm test
	go test -race -count=1 ./...
	go vet ./...

dev: frontend
	ENABLE_SMTP=false ENABLE_PROXY_SMTP=false go run -mod=readonly .
