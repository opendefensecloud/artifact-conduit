package fuzzer

import (
	"gitlab.opencode.de/bwi/ace/artifact-conduit/api/order"
	"sigs.k8s.io/randfill"

	runtimeserializer "k8s.io/apimachinery/pkg/runtime/serializer"
)

// Funcs returns the fuzzer functions for the apps api group.
var Funcs = func(codecs runtimeserializer.CodecFactory) []interface{} {
	return []interface{}{
		func(s *order.OrderSpec, c randfill.Continue) {
			c.FillNoCustom(s) // fuzz self without calling this function again
		},
	}
}
