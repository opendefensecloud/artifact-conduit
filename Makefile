# Include ODC common make targets
DEV_KIT_VERSION := v2.0.0
-include common.mk
common.mk:
	@[ -f .common.mk-download ] || \
		curl --fail -sSL https://raw.githubusercontent.com/opendefensecloud/dev-kit/$(DEV_KIT_VERSION)/common.mk \
		  -o .common.mk-download
	mv .common.mk-download $@
	printf '%s' '$(DEV_KIT_VERSION)' > .common.mk-version
	touch .common.mk-checked

ARC_CHART_DIR ?= $(BUILD_PATH)/charts/arc

export GOPRIVATE=*.go.opendefense.cloud/arc
export GNOSUMDB=*.go.opendefense.cloud/arc
export GNOPROXY=*.go.opendefense.cloud/arc

APISERVER_IMG ?= apiserver:latest
MANAGER_IMG ?= manager:latest
DOCS_IMG ?= arc-docs:latest

ENVTEST_K8S_VERSION ?= 1.36.1

# Repo branch protection settings
REPO_ADMIN_BYPASS := false
REPO_REQUIRED_APPROVING_REVIEW_COUNT := 1
REPO_REQUIRE_CODE_OWNER_REVIEW := false
REPO_REQUIRE_BRANCH_UP_TO_DATE := true
REPO_STATUS_CHECKS := ["check", "CodeQL"]
REPO_RULESET_BRANCHES := ["release/*"]
REPO_ALLOW_MERGE_COMMIT := true
REPO_ALLOW_SQUASH_MERGE := false
REPO_ALLOW_REBASE_MERGE := false
REPO_REQUIRE_LAST_PUSH_APPROVAL := true

# Kind node image for local/e2e clusters — defaults to track ENVTEST_K8S_VERSION
# so envtest (`make test`) and Kind-based clusters (`make dev-cluster`,
# `make test-e2e`) target the same K8s release. Override KIND_NODE_IMAGE
# directly to decouple them.
KIND_NODE_IMAGE ?= kindest/node:v$(patsubst v%,%,$(ENVTEST_K8S_VERSION))

# The image tag (e.g. v1.36.0) the cluster's K8s server version must match.
# Derived from KIND_NODE_IMAGE itself so overriding the image keeps the check
# honest. Splitting on ':' takes the tag even with a registry:port prefix.
KIND_NODE_VERSION := $(lastword $(subst :, ,$(KIND_NODE_IMAGE)))

export ARGO_WORKFLOWS_VERSION := $(shell awk '/^[ \t]+github.com\/argoproj\/argo-workflows/ {print $$2}' go.mod)
export CERTMANAGER_VERSION := v1.21.1
export TRUSTMANAGER_VERSION := v0.24.0

LICENSE := apache
LICENSE_COMMENT := BWI GmbH and Artifact Conduit contributors

.PHONY: codegen
codegen: $(OPENAPI_GEN) ## Run code generation, e.g. openapi
	OPENAPI_GEN=$(OPENAPI_GEN) ./hack/update-codegen.sh
	$(MAKE) docs-crd-ref
	$(MAKE) docs-helm-ref

.PHONY: fmt
fmt: $(GOLANGCI_LINT) ## Add license headers and format code
	$(MAKE) addlicense license=$(LICENSE) comment='$(LICENSE_COMMENT)' pattern='*\.go'
	$(GO) fmt ./...
	$(GOLANGCI_LINT) run --fix

.PHONY: lint
lint: lint-no-golangci golangci-lint ## Run linters

.PHONY: lint-no-golangci
lint-no-golangci: $(ADDLICENSE) shellcheck  ## Run linters but not golangci-lint to exit early in CI/CD pipeline
	$(MAKE) addlicense-check license=apache comment='$(LICENSE_COMMENT)' pattern='*\.go'

.PHONY: envtest-binaries-sideload
envtest-binaries-sideload: $(SETUP_ENVTEST) ## Populate the envtest cache for ENVTEST_K8S_VERSION from upstream K8s/etcd releases when controller-tools hasn't packaged it
	@SETUP_ENVTEST=$(SETUP_ENVTEST) BIN_DIR=$(LOCALBIN) YQ=$(YQ) \
		bash hack/envtest-sideload.sh $(ENVTEST_K8S_VERSION)

.PHONY: test
test: $(SETUP_ENVTEST) $(GINKGO) envtest-binaries-sideload ## Run all tests
	@KUBEBUILDER_ASSETS="$(shell $(SETUP_ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -i -p path)" $(GINKGO) -r -cover --fail-fast --require-suite -covermode count --output-dir=$(BUILD_PATH) -coverprofile=arc.full.coverprofile $(testargs)
	@cat arc.full.coverprofile | grep -v /arc/api > arc.coverprofile

.PHONY: manifests
manifests: $(CONTROLLER_GEN) ## Generate ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role paths="./pkg/controller/...;./api/..." output:rbac:artifacts:config=charts/arc/files

KIND_CLUSTER_E2E  ?= arc-test-e2e
E2E_IMAGE_SOURCE  ?= local
REGISTRY          ?= localhost/local
TAG               ?= e2e

# Like common.mk's setup-local-cluster, but pins the node image to
# KIND_NODE_IMAGE so Kind clusters track the same K8s release as envtest.
.PHONY: kind-cluster
kind-cluster: ## Create the Kind cluster $(KIND_CLUSTER) pinned to KIND_NODE_IMAGE if it does not exist
	@command -v $(KIND) >/dev/null 2>&1 || { \
		echo "Kind is not installed. Please install Kind manually."; \
		exit 1; \
	}
	@if $(KIND) get clusters 2>/dev/null | grep -qx "$(KIND_CLUSTER)"; then \
		current=$$($(KUBECTL) --context kind-$(KIND_CLUSTER) version -o json 2>/dev/null \
			| $(JQ) -r '.serverVersion.gitVersion // empty' 2>/dev/null) || true; \
		case "$$current" in \
			$(KIND_NODE_VERSION) | $(KIND_NODE_VERSION)[-+]*) \
				echo "Kind cluster '$(KIND_CLUSTER)' already exists at $$current. Skipping creation." ;; \
			*) \
				echo "error: Kind cluster '$(KIND_CLUSTER)' runs '$${current:-unknown}', but $(KIND_NODE_VERSION) is required (KIND_NODE_IMAGE=$(KIND_NODE_IMAGE))." >&2; \
				echo "       Recreate it: $(KIND) delete cluster --name $(KIND_CLUSTER) && $(MAKE) kind-cluster KIND_CLUSTER=$(KIND_CLUSTER)" >&2; \
				exit 1 ;; \
		esac; \
	else \
		echo "Creating Kind cluster '$(KIND_CLUSTER)' ($(KIND_NODE_IMAGE))..."; \
		$(KIND) create cluster --name $(KIND_CLUSTER) --image $(KIND_NODE_IMAGE); \
	fi

.PHONY: test-e2e
test-e2e: manifests ## Run the e2e tests. Expected an isolated environment using Kind.
	@if [ "$(E2E_IMAGE_SOURCE)" = "local" ]; then KIND=$(KIND) bash hack/require-kind-version.sh; fi
	$(MAKE) kind-cluster KIND_CLUSTER=$(KIND_CLUSTER_E2E)
	KIND=$(KIND) \
	KIND_CLUSTER=$(KIND_CLUSTER_E2E) \
	HELM=$(HELM) \
	E2E_IMAGE_SOURCE=$(E2E_IMAGE_SOURCE) \
	REGISTRY=$(REGISTRY) \
	IMAGE_TAG=$(TAG) \
	go test -count=1 -tags=e2e ./test/e2e/ -v -timeout=1h -ginkgo.v -ginkgo.timeout=1h
	$(MAKE) cleanup-test-e2e

.PHONY: cleanup-test-e2e
cleanup-test-e2e: ## Tear down the Kind cluster used for e2e tests
	@$(KIND) delete cluster --name $(KIND_CLUSTER_E2E)


KIND_CLUSTER_DEV ?= arc-dev

.PHONY: dev-cluster
dev-cluster: manifests ## Install all necessary components into local Kind cluster for local development
	$(MAKE) kind-cluster KIND_CLUSTER=$(KIND_CLUSTER_DEV)
	@echo -e "\nSETTING UP CERT-MANAGER:\n"
	$(KUBECTL) apply --context kind-$(KIND_CLUSTER_DEV) -f \
		https://github.com/cert-manager/cert-manager/releases/download/$(CERTMANAGER_VERSION)/cert-manager.yaml
	$(KUBECTL) wait deployment.apps/cert-manager-webhook --for condition=Available --namespace cert-manager --timeout 5m
	$(KUBECTL) apply --context kind-$(KIND_CLUSTER_DEV) -n cert-manager -f \
		test/fixtures/certmanager.yaml

	@echo -e "\nSETTING UP TRUST-MANAGER:\n"
	$(HELM) upgrade --install --namespace=cert-manager trust-manager oci://quay.io/jetstack/charts/trust-manager --version $(TRUSTMANAGER_VERSION)
	$(KUBECTL) wait deployment.apps/trust-manager --for condition=Available --namespace cert-manager --timeout 5m
	$(KUBECTL) apply --context kind-$(KIND_CLUSTER_DEV) -n cert-manager -f  \
		test/fixtures/trustmanager.yaml
	$(KUBECTL) label --context kind-$(KIND_CLUSTER_DEV) namespace default trust=enabled --overwrite

	@echo -e "\nSETTING UP ARGO WORKFLOWS:\n"
	$(KUBECTL) --context kind-$(KIND_CLUSTER_DEV) create namespace argo || true
	$(KUBECTL) apply --context kind-$(KIND_CLUSTER_DEV) -n argo --server-side -f \
		https://github.com/argoproj/argo-workflows/releases/download/$(ARGO_WORKFLOWS_VERSION)/quick-start-minimal.yaml
	$(KUBECTL) patch configmap workflow-controller-configmap --context kind-$(KIND_CLUSTER_DEV) -n argo --type=merge -p \
		'{"data": {"artifactRepository": "archiveLogs: false\n"}}'
	$(KUBECTL) apply --context kind-$(KIND_CLUSTER_DEV) -n default -f \
		test/fixtures/secret.yaml
	$(KUBECTL) apply --context kind-$(KIND_CLUSTER_DEV) -n default -f \
		test/fixtures/service-account.yaml

	@echo -e "\nSETTING UP MINIO:\n"
	$(HELM) upgrade --install --create-namespace --namespace=minio --repo=https://charts.min.io -f test/fixtures/dst-minio.yaml dst minio

	@echo -e "\nSETTING UP ZOT:\n"
	$(HELM) upgrade --install --create-namespace --namespace=zot --repo=https://zotregistry.dev/helm-charts -f test/fixtures/dst-zot.yaml dst zot
	$(KUBECTL) apply --context kind-$(KIND_CLUSTER_DEV) -n zot -f \
		test/fixtures/zot-cert.yaml

	@echo -e "\nSETTING UP ARC:\n"
	$(HELM) upgrade --install --create-namespace \
		--namespace arc-system arc charts/arc \
		--set fullnameOverride=arc \
		--set apiserver.args.cronMinScheduleInterval=30s
	@echo -e "\nDONE"

TIMESTAMP ?= $(shell date '+%Y%m%d%H%M%S')

.PHONY: dev-cluster-rebuild
dev-cluster-rebuild: ## Rebuild local images, load them into Kind cluster and update deployment
	$(MAKE) APISERVER_IMG=local/arc-apiserver:dev.$(TIMESTAMP) docker-build-apiserver
	$(MAKE) MANAGER_IMG=local/arc-controller-manager:dev.$(TIMESTAMP) docker-build-manager
	$(KIND) load docker-image local/arc-apiserver:dev.$(TIMESTAMP) --name $(KIND_CLUSTER_DEV)
	$(KIND) load docker-image local/arc-controller-manager:dev.$(TIMESTAMP) --name $(KIND_CLUSTER_DEV)
	$(HELM) upgrade --namespace arc-system arc charts/arc \
		--set fullnameOverride=arc \
		--set apiserver.image.repository=local/arc-apiserver \
		--set apiserver.image.tag=dev.$(TIMESTAMP) \
		--set controller.image.repository=local/arc-controller-manager \
		--set controller.image.tag=dev.$(TIMESTAMP)

.PHONY: cleanup-dev-cluster
cleanup-dev-cluster: ## Tear down the Kind cluster used for e2e tests
	@$(KIND) delete cluster --name $(KIND_CLUSTER_DEV)

# Docker images

# docker is assumed to be installed on the machine
DOCKER ?= docker

.PHONY: docker-build
docker-build: docker-build-apiserver docker-build-manager ## Build apiserver and manager image

.PHONY: docker-build-apiserver
docker-build-apiserver: ## Build apiserver image
	$(DOCKER) build --target apiserver -t ${APISERVER_IMG} .

.PHONY: docker-build-manager
docker-build-manager: ## Build manager image
	$(DOCKER) build --target manager -t ${MANAGER_IMG} .

.PHONY: docker-build-docs
docker-build-docs: ## Build mkdocs image for local serving of documentation
	@$(DOCKER) build --target mkdocs -t ${DOCS_IMG} .

# Docs

.PHONY: docs-crd-ref
docs-crd-ref: $(CRD_REF_DOCS) ## Generate CRD reference documentation.
	$(CRD_REF_DOCS) --source-path=api/arc/v1alpha1 --config=crd-ref-docs.yaml --output-path=./docs/user-guide/api-reference.md --renderer=markdown

.PHONY: docs-helm-ref
docs-helm-ref: $(HELM_DOCS) ## Generate Helm Chart reference documentation.
	cd $(ARC_CHART_DIR) && $(HELM_DOCS) --template-files=README.md.gotmpl

.PHONY: docs
docs: docs-crd-ref docs-helm-ref docker-build-docs ## Serve the documentation using Docker
	@$(DOCKER) run --rm -it -p 8000:8000 -v ${PWD}:/docs ${DOCS_IMG}
