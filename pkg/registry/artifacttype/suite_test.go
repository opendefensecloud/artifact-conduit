// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package artifacttype_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestArtifactTypeRegistry(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ArtifactType Registry Suite")
}
