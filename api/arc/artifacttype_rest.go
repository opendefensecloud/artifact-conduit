// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package arc

import (
	"go.opendefense.cloud/arc/apiserver/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var _ resource.Object = &ArtifactType{}

func (o *ArtifactType) GetObjectMeta() *metav1.ObjectMeta {
	return &o.ObjectMeta
}

func (o *ArtifactType) NamespaceScoped() bool {
	return true
}

func (o *ArtifactType) New() runtime.Object {
	return &ArtifactType{}
}

func (o *ArtifactType) NewList() runtime.Object {
	return &ArtifactTypeList{}
}

func (o *ArtifactType) GetGroupResource() schema.GroupResource {
	return SchemeGroupVersion.WithResource("artifacttypes").GroupResource()
}

var _ resource.Object = &ClusterArtifactType{}

func (o *ClusterArtifactType) GetObjectMeta() *metav1.ObjectMeta {
	return &o.ObjectMeta
}

func (o *ClusterArtifactType) NamespaceScoped() bool {
	return false
}

func (o *ClusterArtifactType) New() runtime.Object {
	return &ClusterArtifactType{}
}

func (o *ClusterArtifactType) NewList() runtime.Object {
	return &ClusterArtifactTypeList{}
}

func (o *ClusterArtifactType) GetGroupResource() schema.GroupResource {
	return SchemeGroupVersion.WithResource("clusterartifacttypes").GroupResource()
}
