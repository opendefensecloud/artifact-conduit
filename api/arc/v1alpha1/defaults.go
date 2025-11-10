// Copyright 2025 BWI GmbH and Artefact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
)

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	return RegisterDefaults(scheme)
}

// SetDefaults_OrderSpec sets defaults for Order spec
func SetDefaults_OrderSpec(obj *OrderSpec) {
	// ...
}

// SetDefaults_OrderSpec sets defaults for Order spec
func SetDefaults_FragmentSpec(obj *FragmentSpec) {
	// ...
}
