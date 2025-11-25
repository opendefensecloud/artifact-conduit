# API Reference

## Packages
- [arc.bwi.de/v1alpha1](#arcbwidev1alpha1)


## arc.bwi.de/v1alpha1

Package v1alpha1 is the v1alpha1 version of the API.





#### ArtifactTypeRules



ArtifactTypeRules is a set of rules to be used for this type of artifact.



_Appears in:_
- [ArtifactTypeSpec](#artifacttypespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `srcTypes` _string array_ | SrcTypes is a list of Endpoint types, that are supported as source. |  |  |
| `dstTypes` _string array_ | DstTypes is a list of Endpoint types, that are supported as destination. |  |  |


#### ArtifactTypeSpec



ArtifactTypeSpec specifies a type of artifact and describes the corresponding workflow.



_Appears in:_
- [ArtifactType](#artifacttype)
- [ClusterArtifactType](#clusterartifacttype)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `rules` _[ArtifactTypeRules](#artifacttyperules)_ | Rules defines a set of rules for this type. |  |  |
| `parameters` _[ArtifactWorkflowParameter](#artifactworkflowparameter) array_ | Parameters defines extra parameters for the Workflow to use.<br />These parameters will override parameters coming from ArtifactWorkflows. |  |  |
| `workflowTemplateRef` _[ArtifactTypeTemplateRef](#artifacttypetemplateref)_ | WorkflowTemplateRef specifies the corresponding Workflow for this type of artifact. |  |  |




#### ArtifactTypeTemplateRef



ArtifactTypeTemplateRef is used to clearly reference a Argo WorkflowTemplate or ClusterWorkflowTemplate.



_Appears in:_
- [ArtifactTypeSpec](#artifacttypespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the name of the Argo WorkflowTemplate or ClusterWorkflowTemplate. |  |  |
| `clusterScope` _boolean_ | ClusterScope defines whether the name corresponds to Argo WorkflowTemplate or ClusterWorkflowTemplate.<br />For ClusterArtifactType this will always be true and all other values are ignored. |  |  |




#### ArtifactWorkflowParameter



ArtifactWorkflowParameter represents a single key-value parameter pair.



_Appears in:_
- [ArtifactTypeSpec](#artifacttypespec)
- [ArtifactWorkflowSpec](#artifactworkflowspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the key of the parameter. |  |  |
| `value` _string_ | Value is the string value of the parameter. |  |  |


#### ArtifactWorkflowSpec



ArtifactWorkflowSpec specifies a single artifact which is translated into a corresponding Workflow based on its type.



_Appears in:_
- [ArtifactWorkflow](#artifactworkflow)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | Type specifies which ArtifactType is used to process this artifact. |  |  |
| `parameters` _[ArtifactWorkflowParameter](#artifactworkflowparameter) array_ | Parameters defines the key-value pairs, that are passed to the underlying Workflow. |  |  |
| `srcSecretRef` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#localobjectreference-v1-core)_ | SrcSecretRef references the secret containing credentials for the source. |  |  |
| `dstSecretRef` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#localobjectreference-v1-core)_ | DstSecretRef references the secret containing credentials for the destination. |  |  |






#### Endpoint



Endpoint is the Schema for the endpoints API



_Appears in:_
- [EndpointList](#endpointlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[EndpointSpec](#endpointspec)_ |  |  |  |




#### EndpointSpec



EndpointSpec specifies a single artifact which is translated into a corresponding Workflow based on its type.



_Appears in:_
- [Endpoint](#endpoint)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | Type specifies which ArtifactType is used to process this artifact. |  |  |
| `remoteURL` _string_ | RemoteURL defines the URL which is used to interact with the endpoint. |  |  |
| `secretRef` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#localobjectreference-v1-core)_ | SecretRef specifies the secret containing the relevant credentials for the endpoint. |  |  |
| `usage` _[EndpointUsage](#endpointusage)_ | Usage defines how the endpoint is allowed to be used. |  |  |




#### EndpointUsage

_Underlying type:_ _string_

EndpointUsage is the usage of the endpoint.



_Appears in:_
- [EndpointSpec](#endpointspec)

| Field | Description |
| --- | --- |
| `PullOnly` | EndpointUsagePullOnly means the endpoint can only be used to pull data.<br /> |
| `PushOnly` | EndpointUsagePushOnly means the endpoint can only be used to push data.<br /> |
| `All` | EndpointUsageAll means the endpoint can be used with all kinds of usage patterns.<br /> |




#### OrderArtifact



OrderArtifact specifies a single artifact which is translated into a corresponding OrderArtifactWorkflow



_Appears in:_
- [OrderSpec](#orderspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | Type specifies which ArtifactType is used to process this artifact. |  |  |
| `srcRef` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#localobjectreference-v1-core)_ | SrcRef defines which Endpoint object is used as source (falls back to OrderDefaults). |  |  |
| `dstRef` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#localobjectreference-v1-core)_ | SrcRef defines which Endpoint object is used as destination (falls back to OrderDefaults). |  |  |
| `spec` _[RawExtension](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#rawextension-runtime-pkg)_ | Spec specifies parameters used by the underlying Workflow. |  |  |


#### OrderArtifactWorkflowStatus







_Appears in:_
- [OrderStatus](#orderstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `artifactIndex` _integer_ | ArtifactIndex references back the index the corresponding artifact has in the .Spec |  |  |
| `phase` _[WorkflowPhase](#workflowphase)_ | Phase tracks which phase the corresponding Workflow is in |  |  |
| `message` _string_ | A human readable message describing the current condition of the artifact workflow. |  |  |


#### OrderDefaults



OrderDefaults is used to set defaults for all other artifacts of an Order.



_Appears in:_
- [OrderSpec](#orderspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `srcRef` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#localobjectreference-v1-core)_ | SrcRef defines which Endpoint object is used as fallback source by all artifacts. |  |  |
| `dstRef` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#localobjectreference-v1-core)_ | DstRef defines which Endpoint object is used as fallback destination by all artifacts. |  |  |


#### OrderSpec



OrderSpec defines the desired state of Order



_Appears in:_
- [Order](#order)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `defaults` _[OrderDefaults](#orderdefaults)_ | Defaults sets up defaults for all artifacts. |  |  |
| `artifacts` _[OrderArtifact](#orderartifact) array_ | Artifacts lists all artifacts, that will be processed by this Order. |  |  |




#### WorkflowPhase

_Underlying type:_ _string_

WorkflowPhase is an enum tracking in which phase a Workflow can be.



_Appears in:_
- [ArtifactWorkflowStatus](#artifactworkflowstatus)
- [OrderArtifactWorkflowStatus](#orderartifactworkflowstatus)

| Field | Description |
| --- | --- |
| `` |  |
| `Pending` |  |
| `Running` |  |
| `Succeeded` |  |
| `Failed` |  |
| `Error` |  |


