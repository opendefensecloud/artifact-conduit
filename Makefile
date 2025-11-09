
# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

# Set MAKEFLAGS to suppress entering/leaving directory messages
MAKEFLAGS += --no-print-directory

BUILD_PATH ?= $(shell pwd)
HACK_DIR ?= $(shell cd hack 2>/dev/null && pwd)
LOCALBIN ?= $(BUILD_PATH)/bin

GO ?= go
GINKGO ?= $(LOCALBIN)/ginkgo
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint
SETUP_ENVTEST ?= $(LOCALBIN)/setup-envtest
ADDLICENSE ?= $(LOCALBIN)/addlicense
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen

GINKGO_VERSION ?= v2.27.2
GOLANGCI_LINT_VERSION ?= v2.5.0
SETUP_ENVTEST_VERSION ?= release-0.22
ADDLICENSE_VERSION ?= v1.1.1
CONTROLLER_TOOLS_VERSION ?= v0.19.0
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


.PHONY: clean
clean:
	rm -rf $(LOCALBIN)

.PHONY: codegen
codegen: ## Run code generation, e.g. openapi
	./hack/update-codegen.sh

.PHONY: fmt
fmt: addlicense ## Add license headers and format code
	find . -not -path '*/.*' -name '*.go' -exec $(ADDLICENSE) -c 'BWI GmbH and Artefact Conduit contributors' -l apache -s=only {} +
	$(GO) fmt ./...

.PHONY: lint
lint: addlicense golangci-lint ## Run linters such as golangci-lint and addlicence checks
	find . -not -path '*/.*' -name '*.go' -exec $(ADDLICENSE) -check  -l apache -s=only -check {} +
	$(GOLANGCI_LINT) run -v

.PHONY: test
test: setup-envtest ginkgo ## Run all tests
	@KUBEBUILDER_ASSETS="$(shell $(SETUP_ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" $(GINKGO) -r -cover --fail-fast --require-suite -covermode count --output-dir=$(BUILD_PATH) -coverprofile=arc.coverprofile

.PHONY: manifests
manifests: controller-gen ## Generate ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role paths="./pkg/controller/...;./api/..." output:rbac:artifacts:config=config/controller/rbac


$(LOCALBIN):
	mkdir -p $(LOCALBIN)

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	test -s $(LOCALBIN)/golangci-lint && $(LOCALBIN)/golangci-lint --version | grep -q $(GOLANGCI_LINT_VERSION) || \
	GOBIN=$(LOCALBIN) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

.PHONY: ginkgo
ginkgo: $(GINKGO) ## Download setup-envtest locally if necessary.
$(GINKGO): $(LOCALBIN)
	test -s $(LOCALBIN)/ginkgo && $(LOCALBIN)/ginkgo version | greo -q $(subst v,,$(GINKGO_VERSION)) || \
	GOBIN=$(LOCALBIN) go install github.com/onsi/ginkgo/v2/ginkgo@$(GINKGO_VERSION)

.PHONY: addlicense
addlicense: $(ADDLICENSE) ## Download addlicense locally if necessary.
$(ADDLICENSE): $(LOCALBIN)
	test -s $(LOCALBIN)/addlicense || \
	GOBIN=$(LOCALBIN) go install github.com/google/addlicense@$(ADDLICENSE_VERSION)

.PHONY: setup-envtest
setup-envtest: $(SETUP_ENVTEST) ## Download setup-envtest locally if necessary.
$(SETUP_ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@$(SETUP_ENVTEST_VERSION)

