// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package arc

import (
	"go.opendefense.cloud/arc/apiserver/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var _ resource.Object = &Order{}
var _ resource.ObjectWithStatusSubResource = &Order{}

func (o *Order) GetObjectMeta() *metav1.ObjectMeta {
	return &o.ObjectMeta
}

func (o *Order) NamespaceScoped() bool {
	return true
}

func (o *Order) New() runtime.Object {
	return &Order{}
}

func (o *Order) NewList() runtime.Object {
	return &OrderList{}
}

func (o *Order) GetGroupResource() schema.GroupResource {
	return SchemeGroupVersion.WithResource("orders").GroupResource()
}

func (o *Order) CopyStatusTo(obj runtime.Object) {
	if obj, ok := obj.(*Order); ok {
		obj.Status = o.Status
	}
}
