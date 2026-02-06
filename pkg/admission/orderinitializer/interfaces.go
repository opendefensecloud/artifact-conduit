// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package orderinitializer

import (
	"k8s.io/apiserver/pkg/admission"

	informers "go.opendefense.cloud/arc/client-go/informers/externalversions"
)

// WantsInternalOrderInformerFactory defines a function which sets InformerFactory for admission plugins that need it
type WantsInternalOrderInformerFactory interface {
	SetInternalOrderInformerFactory(informers.SharedInformerFactory)
	admission.InitializationValidator
}
