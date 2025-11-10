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

// SetDefaults_FragmentSpec sets defaults for Fragment spec
func SetDefaults_FragmentSpec(obj *FragmentSpec) {
	// ...
}

// SetDefaults_EndpointSpec sets defaults for Endpoint spec
func SetDefaults_EndpointSpec(obj *EndpointSpec) {
	// ...
}

// SetDefaults_ArtifactTypeDefinitionSpec sets defaults for ArtifactTypeDefinition spec
func SetDefaults_ArtifactTypeDefinitionSpec(obj *ArtifactTypeDefinitionSpec) {
	// ...
}
