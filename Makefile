SHELL := /bin/sh

IMAGE ?= sheaft:dev
GO_IMAGE ?= golang:1.23

.PHONY: help build test test-docker docker-build docker-run-sample sample clean

help:
	@echo "Targets:"
	@echo "  docker-build        Build container image with sheaft CLI"
	@echo "  docker-run-sample   Run sample pipeline in container"
	@echo "  test-docker         Run Go tests in container"
	@echo "  sample              Alias for docker-run-sample"
	@echo "  clean               Remove generated output files"

build:
	go build -o bin/sheaft ./cmd/sheaft

test:
	go test ./...

test-docker:
	docker run --rm -v "$$(pwd):/src" -w /src $(GO_IMAGE) go test ./...

docker-build:
	docker build -f build/Dockerfile -t $(IMAGE) .

docker-run-sample:
	mkdir -p examples/outputs/generated
	docker run --rm -v "$$(pwd):/workspace" -w /workspace $(IMAGE) run --model examples/outputs/model.sample.json --policy configs/gate.policy.example.yaml --out-dir examples/outputs/generated --seed 42

sample: docker-run-sample

clean:
	rm -rf examples/outputs/generated
