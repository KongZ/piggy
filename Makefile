# Project variables
PACKAGE = github.com/KongZ/piggy
DOCKER_REGISTRY ?= ghcr.io/kongz
PIGGY_ENV_DOCKER_IMAGE = ${DOCKER_REGISTRY}/piggy-env
PIGGY_WEBHOOK_DOCKER_IMAGE = ${DOCKER_REGISTRY}/piggy-webhooks

# Build variables
VERSION = $(shell git describe --tags --always --dirty)
COMMIT_HASH = $(shell git rev-parse --short HEAD 2>/dev/null)
BUILD_DATE = $(shell date +%FT%T%z)
LDFLAGS += -w -s -X main.version=${VERSION} -X main.commitHash=${COMMIT_HASH} -X main.buildDate=${BUILD_DATE}
export CGO_ENABLED ?= 1
export GOOS = $(shell go env GOOS)
# export GO111MODULE=off
ifeq (${VERBOSE}, 1)
	GOARGS += -v
endif

# Docker variables
DOCKER_TAG ?= ${VERSION}

.PHONY: build
build: ## Build all binaries
	@${MAKE} build-piggy-env
	@${MAKE} build-piggy-webhooks

.PHONY: build-piggy-env
build-piggy-env: ## Build a piggy-env binary
	@cd piggy-env && go build ${GOARGS} -tags "${GOTAGS}" -ldflags "${LDFLAGS}" .

.PHONY: build-piggy-webhooks
build-piggy-webhooks: ## Build a piggy-webhooks binary
	@cd piggy-webhooks && go build ${GOARGS} -tags "${GOTAGS}" -ldflags "${LDFLAGS}" .

.PHONY: build-debug
build-debug: GOARGS += -gcflags "all=-N -l"
build-debug: build ## Build a binary with remote debugging capabilities

.PHONY: docker-piggy-env
docker-piggy-env: ## Build a piggy-env Docker image
	docker build -t ${PIGGY_ENV_DOCKER_IMAGE}:${DOCKER_TAG} \
		--build-arg=VERSION=$(VERSION) \
		--build-arg=COMMIT_HASH=$(COMMIT_HASH) \
		--build-arg=BUILD_DATE=$(BUILD_DATE) \
		-f piggy-env/Dockerfile piggy-env

.PHONY: docker-piggy-webhooks
docker-piggy-webhooks: ## Build a piggy-webhooks Docker image
	docker build -t ${PIGGY_WEBHOOK_DOCKER_IMAGE}:${DOCKER_TAG} \
		--build-arg=VERSION=$(VERSION) \
		--build-arg=COMMIT_HASH=$(COMMIT_HASH) \
		--build-arg=BUILD_DATE=$(BUILD_DATE) \
		-f piggy-webhooks/Dockerfile piggy-webhooks

release-%: ## Release a new version
	git tag -m 'Release $*' $*

	@echo "Version updated to $*!"
	@echo
	@echo "To push the changes execute the following:"
	@echo
	@echo "git push; git push origin $*"

.PHONY: patch
patch: ## Release a new patch version
	@${MAKE} release-$(shell git describe --abbrev=0 --tags | awk -F'[ .]' '{print $$1"."$$2"."$$3+1}')

.PHONY: minor
minor: ## Release a new minor version
	@${MAKE} release-$(shell git describe --abbrev=0 --tags | awk -F'[ .]' '{print $$1"."$$2+1".0"}')

.PHONY: major
major: ## Release a new major version
	@${MAKE} release-$(shell git describe --abbrev=0 --tags | awk -F'[ .]' '{print $$1+1".0.0"}')

.PHONY: run ## Run the piggy-webhooks locally
run:
	@cd piggy-webhooks && LISTEN_ADDRESS=:8080 go run .

.PHONY: help
.DEFAULT_GOAL := help
help: # A Self-Documenting Makefile: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
