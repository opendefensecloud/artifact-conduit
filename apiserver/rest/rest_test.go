// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package rest

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("GetAttrs and SelectableFields", func() {
	It("should extract labels and fields from a resource.Object", func() {
		obj := &testObj{}
		obj.SetLabels(map[string]string{"foo": "bar"})
		obj.Name = "myname"
		obj.Namespace = "ns"
		labelsSet, fieldsSet, err := GetAttrs(obj)
		Expect(err).ToNot(HaveOccurred())
		Expect(labelsSet).To(HaveKeyWithValue("foo", "bar"))
		Expect(fieldsSet).To(HaveKeyWithValue("metadata.name", "myname"))
		Expect(fieldsSet).To(HaveKeyWithValue("metadata.namespace", "ns"))
	})

	It("SelectableFields should return correct fields from ObjectMeta", func() {
		meta := &metav1.ObjectMeta{Name: "n", Namespace: "ns", Labels: map[string]string{"x": "y"}}
		fieldsSet := SelectableFields(meta)
		Expect(fieldsSet).To(HaveKeyWithValue("metadata.name", "n"))
		Expect(fieldsSet).To(HaveKeyWithValue("metadata.namespace", "ns"))
	})
})
