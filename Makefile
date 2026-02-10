.PHONY: build-edge build-cp test lint proto

build-edge:
	go build -o bin/edge ./cmd/edge

build-cp:
	go build -o bin/control-plane ./cmd/control-plane

test:
	go test ./...

lint:
	go vet ./...

proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/proto/controlplane/v1/controlplane.proto
