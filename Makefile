.PHONY: all build test vet fmt lint clean docker

BINARY := translate-mcp
CMD := ./cmd/$(BINARY)

all: fmt vet test build

build:
	CGO_ENABLED=0 go build -o $(BINARY) $(CMD)

test:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

lint: fmt vet

clean:
	rm -f $(BINARY)
	docker rmi -f ghcr.io/mohammadraufzahed/translate-mcp:latest 2>/dev/null || true

docker:
	docker build -t ghcr.io/mohammadraufzahed/translate-mcp:latest .
