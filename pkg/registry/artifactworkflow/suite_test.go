// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package artifactworkflow_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestArtifactWorkflowRegistry(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ArtifactWorkflow Registry Suite")
}
