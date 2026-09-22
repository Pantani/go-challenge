# go-challenge Makefile
#
# Every directory containing a go.mod is treated as a module. Each action is
# available for all modules at once (e.g. `make test`) or for a single module
# (e.g. `make test-filestore`).
#
# Any target can run inside Docker by prefixing it with `docker-`
# (e.g. `make docker-test`, `make docker-lint-comms`). If `go` is not installed
# on the host, targets are delegated to Docker automatically.

SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := help

MODULES := $(patsubst %/go.mod,%,$(wildcard */go.mod))

# Tooling
GO            ?= go
GOFMT         ?= gofmt
GOLANGCI_LINT ?= golangci-lint

# Extra flags, e.g. `make test TESTFLAGS="-v -run TestCopy"`
TESTFLAGS ?=
LINTFLAGS ?=

COMPLEXITY_CONFIG := .golangci-complexity.yml

CACHE_DIR := .cache
COVER_DIR := $(CACHE_DIR)/coverage
MIN_COVERAGE ?= 90
COVERAGE_ENV_carrierproxy := CARRIERPROXY_USERNAME=ci-user CARRIERPROXY_PASSWORD=ci-pass

# Docker
DOCKER             ?= docker
GOLANGCI_VERSION   ?= v2.13.2
DOCKER_IMAGE       ?= golangci/golangci-lint:$(GOLANGCI_VERSION)
DOCKER_WORKDIR     := /workspace
DOCKER_RUN_FLAGS   ?= --rm $(if $(shell [ -t 0 ] && echo tty),-it)
DOCKER_RUN = $(DOCKER) run $(DOCKER_RUN_FLAGS) \
	--user $(shell id -u):$(shell id -g) \
	-v "$(CURDIR)":$(DOCKER_WORKDIR) \
	-w $(DOCKER_WORKDIR) \
	-e HOME=/tmp \
	-e IN_DOCKER=1 \
	-e GOCACHE=$(DOCKER_WORKDIR)/$(CACHE_DIR)/go-build \
	-e GOMODCACHE=$(DOCKER_WORKDIR)/$(CACHE_DIR)/go-mod \
	-e GOLANGCI_LINT_CACHE=$(DOCKER_WORKDIR)/$(CACHE_DIR)/golangci-lint \
	$(DOCKER_IMAGE)

# Fall back to Docker when Go is missing on the host.
ifeq ($(IN_DOCKER),)
ifeq ($(shell command -v $(GO) 2>/dev/null),)
AUTO_DOCKER := 1
endif
endif

##@ Help

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "Usage: make \033[36m<target>\033[0m\n"} \
		/^[a-zA-Z0-9_%-]+:.*?##/ { printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2 } \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' $(MAKEFILE_LIST)
	@printf "\n\033[1mModules\033[0m\n  %s\n" "$(MODULES)"
	@printf "\n\033[1mPer-module targets\033[0m\n"
	@printf "  <action>-<module>, e.g. make test-filestore, make lint-comms\n"
	@printf "  actions: test race cover coverage-check vet lint complexity fmt fmt-check tidy tidy-check build check\n"
	@printf "\n\033[1mExamples\033[0m\n"
	@printf "  make test TESTFLAGS=\"-v -run TestCopy\"\n"
	@printf "  make docker-check-notifier\n"
	@$(if $(AUTO_DOCKER),printf "\n\033[33mgo not found on host: targets will run inside Docker ($(DOCKER_IMAGE))\033[0m\n")

##@ Docker

.PHONY: docker-pull docker-shell
docker-pull: ## Pull the Docker image used by docker-* targets
	$(DOCKER) pull $(DOCKER_IMAGE)

docker-shell: ## Open a shell in the Docker toolchain container
	$(DOCKER_RUN) bash

docker-%: ## Run any target inside Docker (e.g. make docker-test-comms)
	$(DOCKER_RUN) make $* TESTFLAGS='$(TESTFLAGS)' LINTFLAGS='$(LINTFLAGS)'

ifeq ($(AUTO_DOCKER),1)

# Go is not available: delegate every other target to Docker.
Makefile: ;
%:
	@echo ">> go not found on host, running 'make $@' in Docker"
	@$(MAKE) --no-print-directory docker-$@

else

##@ All modules

.PHONY: test race cover coverage-check vet lint complexity fmt fmt-check tidy tidy-check build check clean

test: $(addprefix test-,$(MODULES)) ## Run tests for all modules
race: $(addprefix race-,$(MODULES)) ## Run tests with the race detector and coverage for all modules (as CI)
cover: $(addprefix cover-,$(MODULES)) ## Generate coverage reports for all modules
coverage-check: $(addprefix coverage-check-,$(MODULES)) ## Run race tests and enforce minimum coverage for all modules
vet: $(addprefix vet-,$(MODULES)) ## Run go vet for all modules
lint: $(addprefix lint-,$(MODULES)) ## Run golangci-lint for all modules
complexity: $(addprefix complexity-,$(MODULES)) ## Check cyclomatic/cognitive complexity for all modules
fmt: $(addprefix fmt-,$(MODULES)) ## Format code with gofmt for all modules
fmt-check: $(addprefix fmt-check-,$(MODULES)) ## Fail if any module has unformatted files
tidy: $(addprefix tidy-,$(MODULES)) ## Run go mod tidy for all modules
tidy-check: $(addprefix tidy-check-,$(MODULES)) ## Fail if any go.mod/go.sum is not tidy
build: $(addprefix build-,$(MODULES)) ## Build all modules
check: fmt-check tidy-check build vet lint complexity coverage-check ## Run every check (mirrors CI)

clean: ## Remove coverage reports and Docker caches
	@# the Go module cache is read-only; make it writable before removing
	@[ ! -d $(CACHE_DIR) ] || chmod -R u+w $(CACHE_DIR)
	rm -rf $(CACHE_DIR)

# Per-module targets, generated for every module in $(MODULES).
define MODULE_RULES
.PHONY: test-$(1) race-$(1) cover-$(1) coverage-check-$(1) vet-$(1) lint-$(1) complexity-$(1) fmt-$(1) fmt-check-$(1) tidy-$(1) tidy-check-$(1) build-$(1) check-$(1)

test-$(1):
	@echo ">> test $(1)"
	cd $(1) && $(GO) test $(TESTFLAGS) ./...

race-$(1):
	@echo ">> race $(1)"
	cd $(1) && CGO_ENABLED=1 $(GO) test -race -cover $(TESTFLAGS) ./...

coverage-check-$(1):
	@echo ">> coverage-check $(1)"
	@mkdir -p $(COVER_DIR)
	cd $(1) && $(COVERAGE_ENV_$(1)) CGO_ENABLED=1 $(GO) test -race $(TESTFLAGS) -coverprofile=$(CURDIR)/$(COVER_DIR)/$(1).out ./...
	cd $(1) && GO=$(GO) $(CURDIR)/scripts/check-coverage.sh $(CURDIR)/$(COVER_DIR)/$(1).out $(MIN_COVERAGE)

cover-$(1):
	@echo ">> cover $(1)"
	@mkdir -p $(COVER_DIR)
	cd $(1) && $(GO) test $(TESTFLAGS) -covermode=atomic -coverprofile=$(CURDIR)/$(COVER_DIR)/$(1).out ./...
	cd $(1) && $(GO) tool cover -html=$(CURDIR)/$(COVER_DIR)/$(1).out -o $(CURDIR)/$(COVER_DIR)/$(1).html
	@cd $(1) && $(GO) tool cover -func=$(CURDIR)/$(COVER_DIR)/$(1).out | tail -1
	@echo "   html report: $(COVER_DIR)/$(1).html"

vet-$(1):
	@echo ">> vet $(1)"
	cd $(1) && $(GO) vet ./...

lint-$(1):
	@echo ">> lint $(1)"
	@command -v $(GOLANGCI_LINT) >/dev/null || { echo "$(GOLANGCI_LINT) not found; install it or run 'make docker-lint-$(1)'"; exit 1; }
	cd $(1) && $(GOLANGCI_LINT) run $(LINTFLAGS) ./...

complexity-$(1):
	@echo ">> complexity $(1)"
	@command -v $(GOLANGCI_LINT) >/dev/null || { echo "$(GOLANGCI_LINT) not found; install it or run 'make docker-complexity-$(1)'"; exit 1; }
	cd $(1) && $(GOLANGCI_LINT) run --config $(CURDIR)/$(COMPLEXITY_CONFIG) $(LINTFLAGS) ./...

fmt-$(1):
	@echo ">> fmt $(1)"
	$(GOFMT) -s -w $(1)

fmt-check-$(1):
	@echo ">> fmt-check $(1)"
	@out="$$$$($(GOFMT) -s -l $(1))"; \
	if [ -n "$$$$out" ]; then echo "unformatted files (run 'make fmt-$(1)'):"; echo "$$$$out"; exit 1; fi

tidy-$(1):
	@echo ">> tidy $(1)"
	cd $(1) && $(GO) mod tidy

tidy-check-$(1):
	@echo ">> tidy-check $(1)"
	@# portable alternative to `go mod tidy -diff` (Go 1.23+): tidy, diff against a backup, restore
	@cd $(1) && bak="$$$$(mktemp -d)" && cp go.mod "$$$$bak/" && { [ ! -f go.sum ] || cp go.sum "$$$$bak/"; } && \
	status=0 && $(GO) mod tidy || status=$$$$?; \
	if [ "$$$$status" -eq 0 ]; then \
		diff -u "$$$$bak/go.mod" go.mod || status=1; \
		if [ -f "$$$$bak/go.sum" ] || [ -f go.sum ]; then diff -u "$$$$bak/go.sum" go.sum 2>/dev/null || { [ ! -f "$$$$bak/go.sum" ] && [ ! -s go.sum ]; } || status=1; fi; \
	fi; \
	cp "$$$$bak/go.mod" go.mod; \
	if [ -f "$$$$bak/go.sum" ]; then cp "$$$$bak/go.sum" go.sum; else rm -f go.sum; fi; \
	rm -rf "$$$$bak"; \
	[ "$$$$status" -eq 0 ] || echo "go.mod/go.sum not tidy (run 'make tidy-$(1)')"; \
	exit "$$$$status"

build-$(1):
	@echo ">> build $(1)"
	cd $(1) && $(GO) build ./...

check-$(1): fmt-check-$(1) tidy-check-$(1) build-$(1) vet-$(1) lint-$(1) complexity-$(1) coverage-check-$(1)
endef

$(foreach m,$(MODULES),$(eval $(call MODULE_RULES,$(m))))

endif
