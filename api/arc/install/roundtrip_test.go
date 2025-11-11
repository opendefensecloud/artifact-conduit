// Copyright 2025 BWI GmbH and Artefact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"testing"

	"github.com/opendefensecloud/artifact-conduit/api/arc/fuzzer"
	"k8s.io/apimachinery/pkg/api/apitesting/roundtrip"
)

func TestRoundTripTypes(t *testing.T) {
	roundtrip.RoundTripTestForAPIGroup(t, Install, fuzzer.Funcs)
	// TODO: enable protobuf generation for the sample-apiserver
	// roundtrip.RoundTripProtobufTestForAPIGroup(t, Install, orderfuzzer.Funcs)
}
