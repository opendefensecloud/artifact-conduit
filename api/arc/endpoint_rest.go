// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package arc

import (
	"go.opendefense.cloud/arc/apiserver/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var _ resource.Object = &Endpoint{}

func (o *Endpoint) GetObjectMeta() *metav1.ObjectMeta {
	return &o.ObjectMeta
}

func (o *Endpoint) NamespaceScoped() bool {
	return true
}

func (o *Endpoint) New() runtime.Object {
	return &Endpoint{}
}

func (o *Endpoint) NewList() runtime.Object {
	return &EndpointList{}
}

func (o *Endpoint) GetGroupResource() schema.GroupResource {
	return SchemeGroupVersion.WithResource("endpoints").GroupResource()
}
