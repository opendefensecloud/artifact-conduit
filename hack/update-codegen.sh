#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

THIS_PKG="go.opendefense.cloud/arc"

SCRIPT_DIR="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
PROJECT_DIR="$SCRIPT_DIR/.."

(cd "$PROJECT_DIR"; go mod download)

CODEGEN_PKG=$(go list -m -f '{{.Dir}}' k8s.io/code-generator)
# shellcheck disable=SC1091 # we trust kube_codegen.sh
source "${CODEGEN_PKG}/kube_codegen.sh"

kube::codegen::gen_helpers \
    --boilerplate "${SCRIPT_DIR}/boilerplate.go.txt" \
    "${PROJECT_DIR}/api"

# NOTE: unsure why, but openapi-gen opens files not in read-only mode, so let's
#       workaround this for now by setting chmod for relevant modules
#       https://github.com/kubernetes/kubernetes/issues/136295
function cleanup_workaround {
  "${SCRIPT_DIR}/use-local-modules.sh" --restore
}
trap cleanup_workaround EXIT
"${SCRIPT_DIR}/use-local-modules.sh" \
  --dir "${SCRIPT_DIR}/../bin/.modules" \
  k8s.io/api=https://github.com/kubernetes/api.git \
  k8s.io/apimachinery=https://github.com/kubernetes/apimachinery.git
go mod tidy

kube::codegen::gen_openapi \
    --output-dir "${PROJECT_DIR}/client-go/openapi" \
    --output-pkg "${THIS_PKG}/client-go/openapi" \
    --report-filename "$PROJECT_DIR/client-go/openapi/api_violations.report" --update-report \
    --output-model-name-file "zz_generated.model_name.go" \
    --boilerplate "${PROJECT_DIR}/hack/boilerplate.go.txt" \
    --extra-pkgs "k8s.io/api/core/v1" \
    "${PROJECT_DIR}/api"

kube::codegen::gen_client \
  --with-watch \
  --with-applyconfig \
  --applyconfig-name "applyconfigurations" \
  --clientset-name "clientset" \
  --listers-name "listers" \
  --informers-name "informers" \
  --output-dir "$PROJECT_DIR/client-go" \
  --output-pkg "${THIS_PKG}/client-go" \
  --boilerplate "$SCRIPT_DIR/boilerplate.go.txt" \
  "$PROJECT_DIR/api"
