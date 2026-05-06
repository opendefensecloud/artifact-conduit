// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	wfv1alpha1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	arcv1alpha1 "go.opendefense.cloud/arc/api/arc/v1alpha1"
)

func generatePodNameFromNodeStatus(node wfv1alpha1.NodeStatus) string {
	podId := node.ID[strings.LastIndex(node.ID, "-")+1:]
	return fmt.Sprintf("%s-%s-%s", node.BoundaryID, node.TemplateName, podId)
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

func cloneObjectMeta(meta metav1.ObjectMeta, name string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: meta.Namespace,
		Name:      name,
		Labels:    meta.Labels,
	}
}

func awObjectMeta(order *arcv1alpha1.Order, sha string) metav1.ObjectMeta {
	return cloneObjectMeta(order.ObjectMeta, awName(order, sha))
}

func workflowObjectMeta(aw *arcv1alpha1.ArtifactWorkflow) metav1.ObjectMeta {
	return cloneObjectMeta(aw.ObjectMeta, aw.Name)
}

// TODO: add unit tests
func dawToParameters(daw *desiredAW) ([]arcv1alpha1.ArtifactWorkflowParameter, error) {
	// Add permanent parameters
	params := map[string]string{}
	params["srcType"] = daw.srcEndpoint.Spec.Type
	params["srcRemoteURL"] = daw.srcEndpoint.Spec.RemoteURL
	params["dstType"] = daw.dstEndpoint.Spec.Type
	params["dstRemoteURL"] = daw.dstEndpoint.Spec.RemoteURL
	params["srcSecret"] = fmt.Sprintf("%v", daw.srcEndpoint.Spec.SecretRef.Name != "")
	params["dstSecret"] = fmt.Sprintf("%v", daw.dstEndpoint.Spec.SecretRef.Name != "")
	params["cron"] = fmt.Sprintf("%v", daw.cron != nil)

	// Add parameters coming from artifact spec
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

	for k, v := range flattened {
		params[k] = fmt.Sprintf("%v", v)
	}

	// Add parameters coming from type (can override existing)
	for _, p := range daw.typeSpec.Parameters {
		params[p.Name] = p.Value
	}

	// Finally we convert it to a list
	list := make([]arcv1alpha1.ArtifactWorkflowParameter, 0, len(params))
	for name, value := range params {
		list = append(list, arcv1alpha1.ArtifactWorkflowParameter{
			Name:  name,
			Value: value,
		})
	}

	return list, nil
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

// GetForceAtAnnotationValue returns the time specified in the force reconcile annotation, or zero time if not present.
func GetForceAtAnnotationValue(o metav1.Object) (time.Time, error) {
	annotations := o.GetAnnotations()
	if forceAt, present := annotations[AnnotationForceAt]; present {
		i, err := strconv.ParseInt(forceAt, 10, 64)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid force reconcile annotation: %w", err)
		}
		forceAtTime := time.Unix(i, 0)

		return forceAtTime, nil
	}

	return time.Time{}, nil
}
