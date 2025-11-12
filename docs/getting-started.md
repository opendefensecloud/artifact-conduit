# Getting started

Users interact with ARC primarily through the `arcctl` CLI tool, which provides commands for artifact operations and resource management. The CLI communicates with the Kubernetes API Server, which forwards ARC-specific requests to the ARC API Server extension. This creates a declarative workflow where users define desired artifact states, and the system executes the necessary operations.
