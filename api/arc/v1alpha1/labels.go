// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

const (
	// LabelArtifactType records which ArtifactType an ArtifactWorkflow was
	// derived from. The type is only present on the owning Order, so the
	// controller stamps it here to make workflows selectable and observable
	// by type.
	LabelArtifactType = "arc.opendefense.cloud/artifact-type"
)
