// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package controller

const (
	AnnotationRequestedAt = "arc.opendefense.cloud/requested-at"
	AnnotationForceAt     = "arc.opendefense.cloud/force-at"
)

// Event reasons. These double as the reason label on arc_reconcile_errors_total,
// so the metric and the Kubernetes Event for the same failure always agree.
const (
	ReasonInvalid             = "Invalid"
	ReasonInvalidEndpoint     = "InvalidEndpoint"
	ReasonInvalidArtifactType = "InvalidArtifactType"
	ReasonInvalidSecret       = "InvalidSecret"
	ReasonComputationFailed   = "ComputationFailed"
	ReasonHydrationFailed     = "HydrationFailed"
	ReasonCreationFailed      = "CreationFailed"
	ReasonDeletionFailed      = "DeletionFailed"
)

// Controller names used as the controller label on arc_reconcile_errors_total.
const (
	ControllerOrder            = "order"
	ControllerArtifactWorkflow = "artifactworkflow"
)

// ReasonDeleting is the Event reason for the informational warning emitted while
// an order's deletion is in progress. It is not a failure, so it is not counted
// on arc_reconcile_errors_total.
const ReasonDeleting = "Deleting"
