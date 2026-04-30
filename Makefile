# Include ODC common make targets
DEV_KIT_VERSION := v1.0.4
-include common.mk
common.mk:
	curl --fail -sSL https://raw.githubusercontent.com/opendefensecloud/dev-kit/$(DEV_KIT_VERSION)/common.mk -o common.mk.download && \
	mv common.mk.download $@

ARC_CHART_DIR ?= $(BUILD_PATH)/charts/arc

export GOPRIVATE=*.go.opendefense.cloud/arc
export GNOSUMDB=*.go.opendefense.cloud/arc
export GNOPROXY=*.go.opendefense.cloud/arc

APISERVER_IMG ?= apiserver:latest
MANAGER_IMG ?= manager:latest
DOCS_IMG ?= arc-docs:latest

ENVTEST_K8S_VERSION ?= 1.36.0

LICENSE := apache
LICENSE_COMMENT := BWI GmbH and Artifact Conduit contributors

.PHONY: codegen
codegen: $(OPENAPI_GEN) ## Run code generation, e.g. openapi
	OPENAPI_GEN=$(OPENAPI_GEN) ./hack/update-codegen.sh
	$(MAKE) docs-crd-ref

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

.PHONY: test
test: $(SETUP_ENVTEST) $(GINKGO) ## Run all tests
	@KUBEBUILDER_ASSETS="$(shell $(SETUP_ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" $(GINKGO) -r -cover --fail-fast --require-suite -covermode count --output-dir=$(BUILD_PATH) -coverprofile=arc.full.coverprofile $(testargs)
	@cat arc.full.coverprofile | grep -v /arc/api > arc.coverprofile

.PHONY: manifests
manifests: $(CONTROLLER_GEN) ## Generate ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role paths="./pkg/controller/...;./api/..." output:rbac:artifacts:config=charts/arc/files

KIND_CLUSTER_E2E ?= arc-test-e2e
.PHONY: test-e2e
test-e2e: manifests ## Run the e2e tests. Expected an isolated environment using Kind.
	$(MAKE) setup-local-cluster KIND_CLUSTER=$(KIND_CLUSTER_E2E)
	KIND=$(KIND) KIND_CLUSTER=$(KIND_CLUSTER_E2E) HELM=$(HELM) go test -tags=e2e ./test/e2e/ -v -timeout=1h -ginkgo.v -ginkgo.timeout=1h
	$(MAKE) cleanup-test-e2e

.PHONY: cleanup-test-e2e
cleanup-test-e2e: ## Tear down the Kind cluster used for e2e tests
	@$(KIND) delete cluster --name $(KIND_CLUSTER_E2E)


KIND_CLUSTER_DEV ?= arc-dev

.PHONY: dev-cluster
dev-cluster: manifests ## Install all necessary components into local Kind cluster for local development
	$(MAKE) setup-local-cluster KIND_CLUSTER=$(KIND_CLUSTER_DEV)
	@echo -e "\nSETTING UP CERT-MANAGER:\n"
	$(KUBECTL) apply --context kind-$(KIND_CLUSTER_DEV) -f \
		https://github.com/cert-manager/cert-manager/releases/download/v1.19.1/cert-manager.yaml
	$(KUBECTL) wait deployment.apps/cert-manager-webhook --for condition=Available --namespace cert-manager --timeout 5m
	$(KUBECTL) apply --context kind-$(KIND_CLUSTER_DEV) -n cert-manager -f \
		test/fixtures/certmanager.yaml

	@echo -e "\nSETTING UP TRUST-MANAGER:\n"
	$(HELM) upgrade --install --namespace=cert-manager trust-manager oci://quay.io/jetstack/charts/trust-manager --version v0.20.2
	$(KUBECTL) wait deployment.apps/trust-manager --for condition=Available --namespace cert-manager --timeout 5m
	$(KUBECTL) apply --context kind-$(KIND_CLUSTER_DEV) -n cert-manager -f  \
		test/fixtures/trustmanager.yaml
	$(KUBECTL) label  --context kind-$(KIND_CLUSTER_DEV) namespace default trust=enabled --overwrite

	@echo -e "\nSETTING UP ARGO WORKFLOWS:\n"
	$(KUBECTL) --context kind-$(KIND_CLUSTER_DEV) create namespace argo || true
	$(KUBECTL) apply --context kind-$(KIND_CLUSTER_DEV) -n argo -f \
		https://github.com/argoproj/argo-workflows/releases/download/v3.7.4/quick-start-minimal.yaml
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
