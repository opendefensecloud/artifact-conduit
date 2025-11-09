#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

THIS_PKG="gitlab.opencode.de/bwi/ace/artifact-conduit"

SCRIPT_DIR="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
PROJECT_DIR="$SCRIPT_DIR/.."

(cd "$PROJECT_DIR"; go mod download)

export GOROOT="${GOROOT:-"$(go env GOROOT)"}"
export GOPATH="${GOPATH:-"$(go env GOPATH)"}"

CODEGEN_PKG=$(go list -m -f '{{.Dir}}' k8s.io/code-generator)
source "${CODEGEN_PKG}/kube_codegen.sh"


function qualify-gvs() {
  APIS_PKG="$1"
  GROUPS_WITH_VERSIONS="$2"
  join_char=""
  res=""

  for GVs in ${GROUPS_WITH_VERSIONS}; do
    IFS=: read -r G Vs <<<"${GVs}"

    for V in ${Vs//,/ }; do
      res="$res$join_char$APIS_PKG/$G/$V"
      join_char=" "
    done
  done

  echo "$res"
}


CLIENT_GROUPS="order"
CLIENT_VERSION_GROUPS="order:v1alpha1"
ALL_VERSION_GROUPS="$CLIENT_VERSION_GROUPS"


kube::codegen::gen_helpers \
    --boilerplate "${SCRIPT_DIR}/boilerplate.go.txt" \
    "${PROJECT_DIR}/api"

# TODO: use kube::codegen::gen_openapi, see commented block below (works and generates same code but exit code != 0)
input_dirs=($(qualify-gvs "${THIS_PKG}/api" "$ALL_VERSION_GROUPS"))
openapi-gen \
    --output-dir "$PROJECT_DIR/client-go/openapi" \
    --output-pkg "${THIS_PKG}/client-go/openapi" \
    --output-file "zz_generated.openapi.go" \
    --report-filename "$PROJECT_DIR/client-go/openapi/api_violations.report" \
    --go-header-file "$SCRIPT_DIR/boilerplate.go.txt" \
    "k8s.io/apimachinery/pkg/apis/meta/v1" \
    "k8s.io/apimachinery/pkg/runtime" \
    "k8s.io/apimachinery/pkg/version" \
    "k8s.io/api/core/v1" \
    "k8s.io/apimachinery/pkg/api/resource" \
    "${input_dirs[@]}"

# kube::codegen::gen_openapi \
#     --output-dir "$PROJECT_DIR/client-go/openapi" \
#     --output-pkg "${THIS_PKG}/client-go/openapi" \
#     --report-filename "$PROJECT_DIR/client-go/openapi/api_violations.report" \
#     --boilerplate "$SCRIPT_DIR/boilerplate.go.txt" \
#     --extra-pkgs "k8s.io/apimachinery/pkg/apis/meta/v1" \
#     --extra-pkgs "k8s.io/apimachinery/pkg/runtime" \
#     --extra-pkgs "k8s.io/apimachinery/pkg/version" \
#     --extra-pkgs "k8s.io/api/core/v1" \
#     --extra-pkgs "k8s.io/apimachinery/pkg/api/resource" \
#     "${PROJECT_DIR}/api"

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
