default: build

build:
	go build -o terraform-provider-qdrant

generate:
	go generate ./...

install: build
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/qdrant/qdrant/0.1.0/darwin_arm64
	cp terraform-provider-qdrant ~/.terraform.d/plugins/registry.terraform.io/qdrant/qdrant/0.1.0/darwin_arm64/terraform-provider-qdrant_v0.1.0

test:
	go test -v ./...

docs:
	go generate ./...

.PHONY: build generate install test docs
