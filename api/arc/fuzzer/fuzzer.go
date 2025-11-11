// Copyright 2025 BWI GmbH and Artefact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package fuzzer

import (
	"github.com/opendefensecloud/artifact-conduit/api/arc"
	"sigs.k8s.io/randfill"

	runtimeserializer "k8s.io/apimachinery/pkg/runtime/serializer"
)

// Funcs returns the fuzzer functions for the apps api group.
var Funcs = func(codecs runtimeserializer.CodecFactory) []interface{} {
	return []interface{}{
		func(s *arc.OrderSpec, c randfill.Continue) {
			c.FillNoCustom(s) // fuzz self without calling this function again
		},
	}
}
