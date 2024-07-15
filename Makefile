.PHONY: build
build:
	go build -mod=vendor -o raft ./cmd/.

.PHONY: local
local:
	./scripts/launch.sh rf-0 8080 0

.PHONY: vendor
vendor:
	go mod tidy
	go mod vendor

%:
	@:

