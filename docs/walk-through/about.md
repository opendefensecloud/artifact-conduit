# About

ARC is implemented as a Kubernetes API Extension Server that provides an API for managing orders.
As a result, ARC `Order` resources can be managed using `kubectl` and natively integrate with other Kubernetes services.

This walk-through builds a small but real use case step by step: transferring a file from an HTTP source into an
S3 bucket. Each step adds one concept on top of the previous one. Progress through the walk-through in sequence to
learn all the basics.

1. [Setup](setup.md): a local cluster with ARC, Argo Workflows and an S3 store.
2. [First transfer](first-transfer.md): the minimal set of resources to move a blob from HTTP to S3.
3. [Verify before you push](verify.md): a multi-step workflow that checks a checksum and scans for viruses.
4. [Conditionals](conditionals.md): pull from HTTP or S3, depending on the source of each artifact.
5. [Schedule transfers](schedule.md): transfer an artifact again and again with a cron schedule.
6. [Next steps](next-steps.md): more examples, Argo features and running ARC in production.

## ARC and Argo Workflows

ARC itself does not move a single byte. It is a smart scheduler for [Argo Workflows](https://argo-workflows.readthedocs.io/en/stable/):

* ARC knows **what** to transfer (`Order`), **from where to where** (`Endpoint`) and **how** (`ArtifactType`).
* ARC validates orders, resolves credentials and creates one Argo `Workflow` per artifact with the right
  parameters and secrets wired in.
* The actual work (pulling, verifying, scanning, pushing) happens in an Argo `WorkflowTemplate` that you write.

This means that to build your own artifact types, you will write Argo Workflows. The walk-through explains the parts
ARC is responsible for and links to the [Argo Workflows documentation](https://argo-workflows.readthedocs.io/en/stable/walk-through/)
for everything else. Getting familiar with Argo Workflows is well worth the time.

For a complete description of the ARC `Order` spec, please refer
to [the spec documentation](../user-guide/api-reference.md#orderspec).
Many more examples are available in
the [`examples` directory](https://github.com/opendefensecloud/artifact-conduit/tree/main/examples) on GitHub.
