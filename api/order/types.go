// Copyright 2025 BWI GmbH and Artefact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package order

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OrderList is a list of Order objects.
type OrderList struct {
	metav1.TypeMeta
	metav1.ListMeta

	Items []Order
}

// OrderSpec is the specification of a Order.
type OrderSpec struct {
	runtime.RawExtension
}

// OrderStatus is the status of a Order.
type OrderStatus struct {
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Order is an example type with a spec and a status.
type Order struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec   OrderSpec
	Status OrderStatus
}
