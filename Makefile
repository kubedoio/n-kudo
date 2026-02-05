.PHONY: build-edge build-cp test lint

build-edge:
	go build -o bin/edge ./cmd/edge

build-cp:
	go build -o bin/control-plane ./cmd/control-plane

test:
	go test ./...

lint:
	go vet ./...
