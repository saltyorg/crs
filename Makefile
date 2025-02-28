.DEFAULT_GOAL  := build
CMD            := crs
TARGET         := $(shell go env GOOS)_$(shell go env GOARCH)
DIST_PATH      := dist
BUILD_PATH     := ${DIST_PATH}/${CMD}_${TARGET}
GO_FILES       := $(shell find . -path ./vendor -prune -or -type f -name '*.go' -print)
GIT_COMMIT     := $(shell git rev-parse --short HEAD)
TIMESTAMP      := $(shell date +%s)
VERSION        ?= 0.0.0-dev
CGO            := 0

# Deps
.PHONY: check_goreleaser
check_goreleaser:
	@command -v goreleaser >/dev/null || (echo "goreleaser is required."; exit 1)

.PHONY: test
test: ## Run tests
	go test ./... -cover -v -race ${GO_PACKAGES}

.PHONY: vendor
vendor: ## Vendor files and tidy go.mod
	go mod vendor
	go mod tidy

.PHONY: vendor_update
vendor_update: ## Update vendor dependencies
	go get -u ./...
	${MAKE} vendor

.PHONY: build
build: vendor ${BUILD_PATH}/${CMD} ## Build application

# Binary
${BUILD_PATH}/${CMD}: ${GO_FILES} go.sum
	@echo "Building for ${TARGET}..." && \
	mkdir -p ${BUILD_PATH} && \
	CGO_ENABLED=${CGO} go build \
		-mod vendor \
		-trimpath \
		-ldflags "-s -w -X github.com/saltyorg/crs/build.Version=${VERSION} -X github.com/saltyorg/crs/build.GitCommit=${GIT_COMMIT} -X github.com/saltyorg/crs/build.Timestamp=${TIMESTAMP}" \
		-o ${BUILD_PATH}/${CMD} \
		./cmd/crs

.PHONY: release
release: check_goreleaser ## Generate a release, but don't publish
	goreleaser --skip-validate --skip-publish --clean

.PHONY: publish
publish: check_goreleaser ## Generate a release, and publish
	goreleaser --clean

.PHONY: snapshot
snapshot: check_goreleaser ## Generate a snapshot release
	goreleaser --snapshot --skip "publish" --clean

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'