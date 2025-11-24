# Core Concepts

This page serves as an introduction to the core concepts of Artficat Conduit (ARC).

## The `Order`

The `Order` is the primary user-facing resource for requesting artifact operations.
It declares intent to procure one or more artifacts with shared configuration defaults.

// TODO: add more information here

## The `Endpoint`

The `Endpoint` defines a source or destination location with credentials.

## The `ArtifactType`

It specifies processing rules and references an Argo `WorkflowTemplate` resource for processing artifact types (e.g., "oci", "helm").

// TODO: add more information here

## The `ArtifactWorkflow`

It represents a single artifact operation decomposed from an Order. Argo Workflows is used to orchestrate the execution of these workflows.

// TODO: add more information here
