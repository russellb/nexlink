#
# If you want to see the full commands, run:
#   NOISY_BUILD=y make
#
ifeq ($(NOISY_BUILD),)
    ECHO_PREFIX=@
    CMD_PREFIX=@
    PIPE_DEV_NULL=> /dev/null 2> /dev/null
else
    ECHO_PREFIX=@\#
    CMD_PREFIX=
    PIPE_DEV_NULL=
endif

NEXLINK_ALL_GO:=$(wildcard cmd/nexlink/*.go)

.PHONY: help
help:
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ All

.PHONY: all
all: nexlink go-lint ## Run linters and build nexlink

.PHONY: clean
clean: ## Clean up build artifacts
	$(ECHO_PREFIX) printf "  %-12s %s\n" "[CLEAN]" "dist"
	$(CMD_PREFIX) rm -rf dist

dist:
	$(CMD_PREFIX) mkdir -p $@

.PHONY: nexlink
nexlink: dist/nexlink ## Build nexlink

dist/nexlink: cmd/nexlink/main.go | dist
	$(ECHO_PREFIX) printf "  %-12s $@\n" "[GO BUILD]"
	$(CMD_PREFIX) go build -o $@ $<

.PHONY: go-lint
go-lint: dist/.go-lint-linux ## Run go linter

.PHONY: go-lint-prereqs
go-lint-prereqs:
	$(CMD_PREFIX) if ! which golangci-lint >/dev/null 2>&1; then \
		echo "Please install golangci-lint." ; \
		echo "See: https://golangci-lint.run/usage/install/#local-installation" ; \
		exit 1 ; \
	fi

dist/.go-lint-%: $(NEXLINK_ALL_GO) | go-lint-prereqs dist
	$(ECHO_PREFIX) printf "  %-12s GOOS=$(word 3,$(subst -, ,$@))\n" "[GO LINT]"
	$(CMD_PREFIX) CGO_ENABLED=0 GOOS=$(word 3,$(subst -, ,$@)) GOARCH=amd64 \
		golangci-lint run --build-tags integration --timeout 5m ./...
	$(CMD_PREFIX) touch $@