.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Build

.PHONY: build
build: ## Build the binary
	go build .

##@ Development

.PHONY: generate
generate: counterfeiter ## Generate test fakes
	go generate ./...

##@ Tools

## Tool Binaries
COUNTERFEITER := $(LOCALBIN)/counterfeiter

## Location to install tools to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

.PHONY: counterfeiter
counterfeiter: $(COUNTERFEITER) ## Install counterfeiter
$(COUNTERFEITER): $(LOCALBIN)
	cd $(LOCALBIN); go build -o . github.com/maxbrunsfeld/counterfeiter/v6
