package orderinitializer

import (
	informers "gitlab.opencode.de/bwi/ace/artifact-conduit/client-go/informers/externalversions"
	"k8s.io/apiserver/pkg/admission"
)

// WantsInternalOrderInformerFactory defines a function which sets InformerFactory for admission plugins that need it
type WantsInternalOrderInformerFactory interface {
	SetInternalOrderInformerFactory(informers.SharedInformerFactory)
	admission.InitializationValidator
}
