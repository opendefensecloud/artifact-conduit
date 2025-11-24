// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	wfv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	arcv1alpha1 "go.opendefense.cloud/arc/api/arc/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func generatePodNameFromNodeStatus(node wfv1alpha1.NodeStatus) string {
	podId := node.ID[strings.LastIndex(node.ID, "-")+1:]
	return fmt.Sprintf("%s-%s-%s", node.BoundaryID, node.DisplayName, podId)
}

func namespacedName(namespace, name string) types.NamespacedName {
	return types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
}

func awName(order *arcv1alpha1.Order, sha string) string {
	return fmt.Sprintf("%s-%s", order.Name, sha)
}

func awObjectMeta(order *arcv1alpha1.Order, sha string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: order.Namespace,
		Name:      awName(order, sha),
	}
}

// TODO: add unit tests
func dawToParameters(daw *desiredAW) ([]arcv1alpha1.ArtifactWorkflowParameter, error) {
	params := []arcv1alpha1.ArtifactWorkflowParameter{
		{
			Name:  paramName("src", "type"),
			Value: daw.srcEndpoint.Spec.Type,
		},
		{
			Name:  paramName("src", "remoteURL"),
			Value: daw.srcEndpoint.Spec.RemoteURL,
		},
		{
			Name:  paramName("dst", "type"),
			Value: daw.dstEndpoint.Spec.Type,
		},
		{
			Name:  paramName("dst", "remoteURL"),
			Value: daw.dstEndpoint.Spec.RemoteURL,
		},
		{
			Name:  "srcSecret",
			Value: fmt.Sprintf("%v", daw.srcEndpoint.Spec.SecretRef.Name != ""),
		},
		{
			Name:  "dstSecret",
			Value: fmt.Sprintf("%v", daw.dstEndpoint.Spec.SecretRef.Name != ""),
		},
	}

	spec := map[string]any{}
	raw := daw.artifact.Spec.Raw
	if len(raw) == 0 {
		raw = []byte("{}")
	}
	if err := json.Unmarshal(raw, &spec); err != nil {
		return nil, err
	}
	flattened := map[string]any{}
	flattenMap("spec", spec, flattened)
	for name, value := range flattened {
		params = append(params, arcv1alpha1.ArtifactWorkflowParameter{
			Name:  name,
			Value: fmt.Sprintf("%v", value),
		})
	}

	return params, nil
}

// TODO: add unit tests
func paramName(prefix, suffix string) string {
	return prefix + strings.ToUpper(suffix[:1]) + suffix[1:]
}

// TODO: add unit tests
func flattenMap(prefix string, src map[string]any, dst map[string]any) {
	for k, v := range src {
		kt := strings.ToUpper(k[:1]) + k[1:]
		switch child := v.(type) {
		case map[string]any:
			flattenMap(prefix+k, child, dst)
		case []any:
			for i, av := range child {
				dst[prefix+kt+strconv.Itoa(i)] = av
			}
		default:
			dst[prefix+kt] = v
		}
	}
}
