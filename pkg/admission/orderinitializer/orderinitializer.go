// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package orderinitializer

import (
	informers "go.opendefense.cloud/arc/client-go/informers/externalversions"
	"k8s.io/apiserver/pkg/admission"
)

type pluginInitializer struct {
	informers informers.SharedInformerFactory
}

var _ admission.PluginInitializer = pluginInitializer{}

// New creates an instance of wardle admission plugins initializer.
func New(informers informers.SharedInformerFactory) pluginInitializer {
	return pluginInitializer{
		informers: informers,
	}
}

// Initialize checks the initialization interfaces implemented by a plugin
// and provide the appropriate initialization data
func (i pluginInitializer) Initialize(plugin admission.Interface) {
	if wants, ok := plugin.(WantsInternalOrderInformerFactory); ok {
		wants.SetInternalOrderInformerFactory(i.informers)
	}
}
