.PHONY: build
build:
	go build -mod=vendor -race -o raft ./cmd/.

.PHONY: local-0
local-0:
	./scripts/launch.sh node-0 8080 0 grpc

.PHONY: local-1
local-1:
	./scripts/launch.sh node-1 8081 1 grpc

.PHONY: local-2
local-2:
	./scripts/launch.sh node-2 8082 2 grpc

.PHONY: generate
generate:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/raft.proto

.PHONY: vendor
vendor:
	go mod tidy
	go mod vendor

%:
	@:

