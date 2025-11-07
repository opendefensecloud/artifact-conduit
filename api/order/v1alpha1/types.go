package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OrderList is a list of Order objects.
type OrderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Items []Order `json:"items" protobuf:"bytes,2,rep,name=items"`
}

type ReferenceType string

const (
	OrderReferenceType = ReferenceType("Order")
)

type OrderSpec struct {
	runtime.RawExtension
}

type OrderStatus struct {
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Order struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   OrderSpec   `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status OrderStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}
