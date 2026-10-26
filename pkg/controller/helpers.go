// Copyright BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"strconv"
	"strings"
	"time"

	wfv1alpha1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"

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

const (
	// awNameMaxLength is the maximum length of an ArtifactWorkflow name.
	//
	// The name is reused verbatim as the Argo Workflow/CronWorkflow name.
	// Argo caps Workflow names at 63 characters (the Kubernetes label value
	// limit) and CronWorkflow names at 52, because it appends
	// "-<unix timestamp>" when creating workflows. Those limits are unexported
	// in argo-workflows/workflow/validate, so they can't be referenced here.
	// Use the stricter cron limit since an Order may switch to a cron schedule.
	awNameMaxLength = 52

	// awNameHashLength is the number of hex characters of the order name hash
	// appended when the order name has to be shortened.
	awNameHashLength = 16
)

func awName(order *arcv1alpha1.Order, sha string) string {
	suffix := "-" + sha
	if len(order.Name)+len(suffix) <= awNameMaxLength {
		return order.Name + suffix
	}

	// Keep the start of the order name for readability and append a hash of
	// the full name so that orders sharing the same prefix stay distinct.
	sum := sha256.Sum256([]byte(order.Name))
	nameHash := hex.EncodeToString(sum[:])[:awNameHashLength]
	keep := max(awNameMaxLength-len(suffix)-awNameHashLength-1, 0)

	// Never join the suffix to a trailing separator: "a.-hash" is not a valid
	// DNS-1123 subdomain because the label after the dot starts with a dash.
	prefix := strings.TrimRight(order.Name[:keep], ".-")

	return prefix + "-" + nameHash + suffix
}

func cloneObjectMeta(meta metav1.ObjectMeta, name string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: meta.Namespace,
		Name:      name,
		Labels:    maps.Clone(meta.Labels),
	}
}

func awObjectMeta(order *arcv1alpha1.Order, sha, artifactType string) metav1.ObjectMeta {
	meta := cloneObjectMeta(order.ObjectMeta, awName(order, sha))

	// Nothing validates OrderArtifact.Type against the 63 character label value
	// limit, and an invalid value would make every create of this
	// ArtifactWorkflow fail permanently. Leaving the label off keeps the
	// workflow creatable and reports it as artifact_type="unknown". Truncating
	// would report a wrong type instead of a missing one.
	if len(validation.IsValidLabelValue(artifactType)) > 0 {
		delete(meta.Labels, arcv1alpha1.LabelArtifactType)
		return meta
	}

	if meta.Labels == nil {
		meta.Labels = map[string]string{}
	}
	meta.Labels[arcv1alpha1.LabelArtifactType] = artifactType

	return meta
}

func workflowObjectMeta(aw *arcv1alpha1.ArtifactWorkflow) metav1.ObjectMeta {
	return cloneObjectMeta(aw.ObjectMeta, aw.Name)
}

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

func flattenMap(prefix string, src map[string]any, dst map[string]any) {
	for k, v := range src {
		kt := strings.ToUpper(k[:1]) + k[1:]
		switch child := v.(type) {
		case map[string]any:
			flattenMap(prefix+kt, child, dst)
		case []any:
			for i, av := range child {
				dst[prefix+kt+strconv.Itoa(i)] = av
			}
		default:
			dst[prefix+kt] = v
		}
	}
}

// earliestRequeue returns the soonest of two requeue durations, treating a
// non-positive value as "no requeue requested".
func earliestRequeue(a, b time.Duration) time.Duration {
	if a <= 0 {
		return b
	}
	if b <= 0 {
		return a
	}

	return min(a, b)
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
		if !recordableForceAt(forceAtTime) {
			return time.Time{}, fmt.Errorf("invalid force reconcile annotation: %d cannot be recorded", i)
		}

		return forceAtTime, nil
	}

	return time.Time{}, nil
}

// recordableForceAt reports whether a force annotation value survives being
// recorded in status. That is a metav1.Time, which serialises to null past year
// 9999, so such a value would read back unhandled on every pass. Asking the
// serialisation rather than bounding the year keeps the two from drifting, in
// UTC because that is the zone it serialises in.
func recordableForceAt(forceAt time.Time) bool {
	return forceAt.Equal(metav1.NewTime(forceAt.UTC()).Rfc3339Copy().Time)
}
