
# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

# Set MAKEFLAGS to suppress entering/leaving directory messages
MAKEFLAGS += --no-print-directory

BUILD_PATH ?= $(shell pwd)
HACK_DIR ?= $(shell cd hack 2>/dev/null && pwd)
GINKGO ?= ginkgo
ENVTEST ?= setup-envtest
ENVTEST_K8S_VERSION ?= 1.34.1

export GOPRIVATE=*.opencode.de
export GNOSUMDB=*.opencode.de
export GNOPROXY=*.opencode.de

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

export
include devenv.mk

LOCALBIN ?= $(BUILD_PATH)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

clean:
	rm -rf $(LOCALBIN)

codegen: ## Run code generation, e.g. openapi
	./hack/update-codegen.sh

fmt: ## Add license headers and format code
	addlicense -c 'BWI GmbH and Artefact Conduit contributors' -l apache -s=only **/*.go
	go fmt ./...

lint: ## Run linters such as golangci-lint and addlicence checks
	addlicense -c 'BWI GmbH and Artefact Conduit contributors' -l apache -s=only -check **/*.go && \
	golangci-lint run -v

test: $(LOCALBIN) ## Run all tests
	@KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" $(GINKGO) -r -cover --fail-fast --require-suite -covermode count --output-dir=$(BUILD_PATH) -coverprofile=arc.coverprofile
