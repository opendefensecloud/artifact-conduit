// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package fuzzer

import (
	"go.opendefense.cloud/arc/api/arc"
	"sigs.k8s.io/randfill"

	runtimeserializer "k8s.io/apimachinery/pkg/runtime/serializer"
)

// Funcs returns the fuzzer functions for the apps api group.
var Funcs = func(codecs runtimeserializer.CodecFactory) []any {
	return []any{
		func(s *arc.OrderSpec, c randfill.Continue) {
			c.FillNoCustom(s) // fuzz self without calling this function again
		},
		func(s *arc.ArtifactType, c randfill.Continue) {
			c.FillNoCustom(s) // fuzz self without calling this function again
			// v1alpha1 doesn't have Status fields yet, so clear them for roundtrip testing
			s.Status = arc.ArtifactTypeStatus{}
		},
		func(s *arc.ClusterArtifactType, c randfill.Continue) {
			c.FillNoCustom(s) // fuzz self without calling this function again
			// v1alpha1 doesn't have Status fields yet, so clear them for roundtrip testing
			s.Status = arc.ArtifactTypeStatus{}
		},
	}
}
