.PHONY: build
build:
	go build -mod=vendor -o raft ./cmd/.

.PHONY: local
local:
	./scripts/launch.sh rf-0 8080 0

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

