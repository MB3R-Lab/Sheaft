ifeq ($(OS),Windows_NT)
EXEEXT := .exe
POSIX_SH ?= C:/Program Files/Git/usr/bin/sh.exe
GIT_BASH ?= C:/Program Files/Git/bin/bash.exe
else
SHELL := /bin/sh
EXEEXT :=
POSIX_SH ?= sh
GIT_BASH ?= bash
endif

IMAGE ?= sheaft:dev
IMAGE_REPOSITORY ?= ghcr.io/mb3r-lab/sheaft
CHART_OCI_REPOSITORY ?= oci://ghcr.io/mb3r-lab/charts
LOCAL_REGISTRY ?= localhost:5000
REGISTRY_PORT ?= 5000
GORELEASER_VERSION ?= v2.14.3
GORELEASER ?= go run github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION)
SYFT_VERSION ?= v1.42.2
GOBIN_DIR ?= $(shell go env GOPATH)/bin
HELM_BIN ?= helm
APP_VERSION ?= 0.0.0-dev
CHART_VERSION ?= $(APP_VERSION)
ifeq ($(OS),Windows_NT)
GIT_COMMIT ?= $(strip $(shell powershell -NoProfile -Command "$$commit = git rev-parse HEAD 2>$$null; if ($$LASTEXITCODE -eq 0 -and $$commit) { $$commit } else { 'unknown' }"))
BUILD_DATE ?= $(strip $(shell powershell -NoProfile -Command "$$buildDate = git log -1 --format=%cI 2>$$null; if ($$LASTEXITCODE -eq 0 -and $$buildDate) { $$buildDate } else { (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ') }"))
else
GIT_COMMIT ?= $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= $(shell git log -1 --format=%cI 2>/dev/null || date -u +"%Y-%m-%dT%H:%M:%SZ")
endif
DIST_DIR ?= dist
PLATFORMS ?= linux/amd64,linux/arm64

.PHONY: help build test lint smoke-examples docker-build docker-run-sample sample compatibility-manifest validate-compatibility-manifest validate-chart default-config-pack release-tools release-build image-dry-run image-local chart-package chart-publish-local release-manifest validate-release-manifest validate-release-assets release-dry-run release-local clean clean-dist

ifeq ($(OS),Windows_NT)
define MKDIR_P
powershell -NoProfile -Command "New-Item -ItemType Directory -Force '$(1)' | Out-Null"
endef

define RM_RF
powershell -NoProfile -Command "if (Test-Path '$(1)') { Remove-Item -Recurse -Force '$(1)' }"
endef
else
define MKDIR_P
mkdir -p $(1)
endef

define RM_RF
rm -rf $(1)
endef
endif

help:
	@echo "Targets:"
	@echo "  build                      Build the sheaft CLI locally"
	@echo "  test                       Run Go tests"
	@echo "  lint                       Run go vet"
	@echo "  smoke-examples             Build the CLI and smoke checked-in examples"
	@echo "  docker-build               Build the local container image"
	@echo "  docker-run-sample          Run sample pipeline in the container image"
	@echo "  compatibility-manifest     Generate compatibility-manifest.json from strict contract pins"
	@echo "  chart-package              Lint and package charts/sheaft into dist/charts"
	@echo "  release-build              Produce binaries, archives, checksums, SBOM, and source archive with GoReleaser"
	@echo "  release-manifest           Generate release-manifest.json from dist metadata"
	@echo "  release-dry-run            Local non-publishing release validation path"
	@echo "  release-local              Push image and chart to a local OCI registry and generate a full release manifest"
	@echo "  clean                      Remove dist and generated example output files"

build:
	$(call MKDIR_P,bin)
	go build -o bin/sheaft$(EXEEXT) ./cmd/sheaft

test:
	go test ./...

lint:
	go vet ./...

smoke-examples:
	$(MAKE) build
ifeq ($(OS),Windows_NT)
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/ci/smoke-examples.ps1 -OutRoot .tmp/smoke-examples -BinPath ./bin/sheaft$(EXEEXT)
else
	"$(POSIX_SH)" scripts/ci/smoke-examples.sh .tmp/smoke-examples ./bin/sheaft$(EXEEXT)
endif

docker-build:
	docker build -f build/Dockerfile -t $(IMAGE) .

docker-run-sample:
	mkdir -p examples/outputs/generated
	docker run --rm -v "$$(pwd):/workspace" -w /workspace $(IMAGE) run --model examples/outputs/model.sample.json --policy configs/gate.policy.example.yaml --out-dir examples/outputs/generated --seed 42

sample: docker-run-sample

compatibility-manifest:
	go run ./cmd/releasectl compatibility-manifest --out compatibility-manifest.json

validate-compatibility-manifest:
	go run ./cmd/releasectl validate-compatibility-manifest --manifest compatibility-manifest.json

validate-chart:
	go run ./cmd/releasectl validate-chart --chart-dir charts/sheaft

default-config-pack:
	go run ./cmd/releasectl package-default-config-pack --version $(APP_VERSION) --out $(DIST_DIR)/sheaft-default-config-pack_$(APP_VERSION).tar.gz --metadata-out $(DIST_DIR)/default-config-pack.json

release-tools:
ifeq ($(OS),Windows_NT)
	set "GOBIN=$(GOBIN_DIR)" && go install github.com/anchore/syft/cmd/syft@$(SYFT_VERSION)
else
	GOBIN=$(GOBIN_DIR) go install github.com/anchore/syft/cmd/syft@$(SYFT_VERSION)
endif

release-build: clean-dist release-tools
ifeq ($(OS),Windows_NT)
	set "PATH=$(GOBIN_DIR);%PATH%" && set "APP_VERSION=$(APP_VERSION)" && $(GORELEASER) release --clean --snapshot --skip=publish
else
	PATH="$(GOBIN_DIR):$$PATH" APP_VERSION=$(APP_VERSION) $(GORELEASER) release --clean --snapshot --skip=publish
endif

image-dry-run:
ifeq ($(OS),Windows_NT)
	set "IMAGE_REPOSITORY=$(IMAGE_REPOSITORY)" && set "APP_VERSION=$(APP_VERSION)" && set "GIT_COMMIT=$(GIT_COMMIT)" && set "BUILD_DATE=$(BUILD_DATE)" && set "PLATFORMS=$(PLATFORMS)" && set "IMAGE_METADATA_OUTPUT=$(DIST_DIR)/image-metadata.json" && set "PUBLISH=false" && "$(GIT_BASH)" -lc "scripts/release/build-image.sh"
else
	IMAGE_REPOSITORY=$(IMAGE_REPOSITORY) APP_VERSION=$(APP_VERSION) GIT_COMMIT=$(GIT_COMMIT) BUILD_DATE=$(BUILD_DATE) PLATFORMS=$(PLATFORMS) IMAGE_METADATA_OUTPUT=$(DIST_DIR)/image-metadata.json PUBLISH=false sh scripts/release/build-image.sh
endif

image-local:
ifeq ($(OS),Windows_NT)
	set "IMAGE_REPOSITORY=$(LOCAL_REGISTRY)/sheaft" && set "APP_VERSION=$(APP_VERSION)" && set "GIT_COMMIT=$(GIT_COMMIT)" && set "BUILD_DATE=$(BUILD_DATE)" && set "PLATFORMS=$(PLATFORMS)" && set "IMAGE_METADATA_OUTPUT=$(DIST_DIR)/image-metadata.json" && set "PUBLISH=true" && "$(GIT_BASH)" -lc "scripts/release/build-image.sh"
else
	IMAGE_REPOSITORY=$(LOCAL_REGISTRY)/sheaft APP_VERSION=$(APP_VERSION) GIT_COMMIT=$(GIT_COMMIT) BUILD_DATE=$(BUILD_DATE) PLATFORMS=$(PLATFORMS) IMAGE_METADATA_OUTPUT=$(DIST_DIR)/image-metadata.json PUBLISH=true sh scripts/release/build-image.sh
endif

chart-package:
ifeq ($(OS),Windows_NT)
	set "HELM_BIN=$(HELM_BIN)" && set "APP_VERSION=$(APP_VERSION)" && set "CHART_VERSION=$(CHART_VERSION)" && set "CHART_OUTPUT_DIR=$(DIST_DIR)/charts" && set "CHART_METADATA_OUTPUT=$(DIST_DIR)/chart-metadata.json" && set "PUBLISH=false" && "$(GIT_BASH)" -lc "scripts/release/package-chart.sh"
else
	HELM_BIN=$(HELM_BIN) APP_VERSION=$(APP_VERSION) CHART_VERSION=$(CHART_VERSION) CHART_OUTPUT_DIR=$(DIST_DIR)/charts CHART_METADATA_OUTPUT=$(DIST_DIR)/chart-metadata.json PUBLISH=false sh scripts/release/package-chart.sh
endif

chart-publish-local:
ifeq ($(OS),Windows_NT)
	set "HELM_BIN=$(HELM_BIN)" && set "APP_VERSION=$(APP_VERSION)" && set "CHART_VERSION=$(CHART_VERSION)" && set "CHART_OUTPUT_DIR=$(DIST_DIR)/charts" && set "CHART_OCI_REPOSITORY=oci://$(LOCAL_REGISTRY)/charts" && set "CHART_METADATA_OUTPUT=$(DIST_DIR)/chart-metadata.json" && set "PUBLISH=true" && "$(GIT_BASH)" -lc "scripts/release/package-chart.sh"
else
	HELM_BIN=$(HELM_BIN) APP_VERSION=$(APP_VERSION) CHART_VERSION=$(CHART_VERSION) CHART_OUTPUT_DIR=$(DIST_DIR)/charts CHART_OCI_REPOSITORY=oci://$(LOCAL_REGISTRY)/charts CHART_METADATA_OUTPUT=$(DIST_DIR)/chart-metadata.json PUBLISH=true sh scripts/release/package-chart.sh
endif

release-manifest:
	go run ./cmd/releasectl release-manifest --dist $(DIST_DIR) --out release-manifest.json --app-version $(APP_VERSION) --git-commit $(GIT_COMMIT) --build-date $(BUILD_DATE) --compatibility-manifest compatibility-manifest.json --default-pack-metadata $(DIST_DIR)/default-config-pack.json --image-metadata $(DIST_DIR)/image-metadata.json --chart-metadata $(DIST_DIR)/chart-metadata.json

validate-release-manifest:
	go run ./cmd/releasectl validate-release-manifest --manifest release-manifest.json

validate-release-assets: validate-compatibility-manifest validate-chart validate-release-manifest

release-dry-run: compatibility-manifest test smoke-examples release-build default-config-pack image-dry-run chart-package release-manifest validate-release-assets

release-local: compatibility-manifest test smoke-examples release-build
ifeq ($(OS),Windows_NT)
	set "REGISTRY_PORT=$(REGISTRY_PORT)" && "$(GIT_BASH)" -lc "scripts/release/ensure-local-registry.sh"
else
	REGISTRY_PORT=$(REGISTRY_PORT) sh scripts/release/ensure-local-registry.sh
endif
	$(MAKE) default-config-pack APP_VERSION=$(APP_VERSION)
	$(MAKE) image-local APP_VERSION=$(APP_VERSION) BUILD_DATE=$(BUILD_DATE) GIT_COMMIT=$(GIT_COMMIT)
	$(MAKE) chart-publish-local APP_VERSION=$(APP_VERSION) CHART_VERSION=$(CHART_VERSION)
	$(MAKE) release-manifest APP_VERSION=$(APP_VERSION) BUILD_DATE=$(BUILD_DATE) GIT_COMMIT=$(GIT_COMMIT)
	$(MAKE) validate-release-assets

clean-dist:
	$(call RM_RF,$(DIST_DIR))

clean: clean-dist
	$(call RM_RF,examples/outputs/generated)
	$(call RM_RF,.tmp)
