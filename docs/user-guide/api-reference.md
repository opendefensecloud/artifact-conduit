# API Reference

## Packages
- [arc.opendefense.cloud/v1alpha1](#arcopendefensecloudv1alpha1)


## arc.opendefense.cloud/v1alpha1

Package v1alpha1 is the v1alpha1 version of the API.



#### ArtifactType



ArtifactType is the Schema for the endpoints API



_Appears in:_
- [ArtifactTypeList](#artifacttypelist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ArtifactTypeSpec](#artifacttypespec)_ |  |  |  |
| `status` _[ArtifactTypeStatus](#artifacttypestatus)_ |  |  |  |




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


#### ArtifactTypeStatus



ArtifactTypeStatus defines the observed state of ArtifactType



_Appears in:_
- [ArtifactType](#artifacttype)
- [ClusterArtifactType](#clusterartifacttype)



#### ArtifactTypeTemplateRef



ArtifactTypeTemplateRef is used to clearly reference a Argo WorkflowTemplate or ClusterWorkflowTemplate.



_Appears in:_
- [ArtifactTypeSpec](#artifacttypespec)
- [ArtifactWorkflowSpec](#artifactworkflowspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the name of the Argo WorkflowTemplate or ClusterWorkflowTemplate. |  |  |
| `clusterScope` _boolean_ | ClusterScope defines whether the name corresponds to Argo WorkflowTemplate or ClusterWorkflowTemplate.<br />For ClusterArtifactType this will always be true and all other values are ignored. |  |  |


#### ArtifactWorkflow



ArtifactWorkflow is the Schema for the artifact workflows API



_Appears in:_
- [ArtifactWorkflowList](#artifactworkflowlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ArtifactWorkflowSpec](#artifactworkflowspec)_ |  |  |  |
| `status` _[ArtifactWorkflowStatus](#artifactworkflowstatus)_ |  |  |  |




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
| `workflowTemplateRef` _[ArtifactTypeTemplateRef](#artifacttypetemplateref)_ | WorkflowTemplateRef specifies the corresponding Workflow for this ArtifactWorkflow as derived from ArtifactType |  |  |
| `parameters` _[ArtifactWorkflowParameter](#artifactworkflowparameter) array_ | Parameters defines the key-value pairs, that are passed to the underlying Workflow. |  |  |
| `srcSecretRef` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#localobjectreference-v1-core)_ | SrcSecretRef references the secret containing credentials for the source. |  |  |
| `dstSecretRef` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#localobjectreference-v1-core)_ | DstSecretRef references the secret containing credentials for the destination. |  |  |
| `cron` _[Cron](#cron)_ | Cron specifies options which determine when the order should be scheduled. |  |  |


#### ArtifactWorkflowStatus



ArtifactWorkflowStatus defines the observed state of ArtifactWorkflow



_Appears in:_
- [ArtifactWorkflow](#artifactworkflow)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `phase` _[WorkflowPhase](#workflowphase)_ | Phase tracks which phase the corresponding Workflow is in |  |  |
| `message` _string_ | A human readable message describing the current condition of the artifact workflow. |  |  |
| `completionTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#time-v1-meta)_ | CompletionTime is the time when the workflow finished |  |  |
| `lastScheduled` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#time-v1-meta)_ | LastScheduled is the last time the workflow was scheduled via cron |  |  |
| `succeeded` _integer_ | Succeeded counts how many times child workflows succeeded |  |  |
| `failed` _integer_ | Failed counts how many times child workflows failed |  |  |
| `lastReconcileAt` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#time-v1-meta)_ | LastReconcileAt is the last time the Order was reconciled |  |  |
| `lastForceAt` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#time-v1-meta)_ | LastForceAt is the last time a force reconciliation was requested |  |  |
| `activeWorkflowRef` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#localobjectreference-v1-core)_ | ActiveWorkflowRef tracks the currently spawned workflow, if cron is used.<br />It resets after a successful or failed run. |  |  |


#### ClusterArtifactType



ArtifactType is the Schema for the endpoints API



_Appears in:_
- [ClusterArtifactTypeList](#clusterartifacttypelist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ArtifactTypeSpec](#artifacttypespec)_ |  |  |  |
| `status` _[ArtifactTypeStatus](#artifacttypestatus)_ |  |  |  |




#### Cron



Cron represents an order's cron schedule.



_Appears in:_
- [ArtifactWorkflowSpec](#artifactworkflowspec)
- [OrderArtifact](#orderartifact)
- [OrderDefaults](#orderdefaults)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `timezone` _string_ | Timezone is the timezone against which the cron schedule will be calculated, e.g. "Asia/Tokyo". Default is machine's local time. |  |  |
| `startingDeadlineSeconds` _integer_ | StartingDeadlineSeconds is the K8s-style deadline that will limit the time a Order will be run after its<br />original scheduled time if it is missed. |  | Minimum: 0 <br /> |
| `schedules` _string array_ | Schedules is a list of schedules to run the Order in Cron format |  | MinItems: 1 <br />items:Pattern: ^(@(yearly\|annually\|monthly\|weekly\|daily\|midnight\|hourly)\|@every\s+([0-9]+(ns\|us\|µs\|ms\|s\|m\|h))+\|([0-9*,/?-]+\s+)\{4\}[0-9*,/?-]+)$ <br /> |
| `when` _string_ | When is an expression that determines if a run should be scheduled. |  |  |


#### Endpoint



Endpoint is the Schema for the endpoints API



_Appears in:_
- [EndpointList](#endpointlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[EndpointSpec](#endpointspec)_ |  |  |  |
| `status` _[EndpointStatus](#endpointstatus)_ |  |  |  |




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


#### EndpointStatus



EndpointStatus defines the observed state of Endpoint



_Appears in:_
- [Endpoint](#endpoint)



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


#### Order



Order is the Schema for the orders API



_Appears in:_
- [OrderList](#orderlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[OrderSpec](#orderspec)_ |  |  |  |
| `status` _[OrderStatus](#orderstatus)_ |  |  |  |


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
| `cron` _[Cron](#cron)_ | Cron specifies options which determine when the order should be scheduled (falls back to OrderDefaults). |  |  |


#### OrderArtifactWorkflowStatus







_Appears in:_
- [OrderStatus](#orderstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `phase` _[WorkflowPhase](#workflowphase)_ | Phase tracks which phase the corresponding Workflow is in |  |  |
| `message` _string_ | A human readable message describing the current condition of the artifact workflow. |  |  |
| `completionTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#time-v1-meta)_ | CompletionTime is the time when the workflow finished |  |  |
| `lastScheduled` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#time-v1-meta)_ | LastScheduled is the last time the workflow was scheduled via cron |  |  |
| `succeeded` _integer_ | Succeeded counts how many times child workflows succeeded |  |  |
| `failed` _integer_ | Failed counts how many times child workflows failed |  |  |
| `artifactIndex` _integer_ | ArtifactIndex references back the index the corresponding artifact has in the .Spec |  |  |


#### OrderDefaults



OrderDefaults is used to set defaults for all other artifacts of an Order.



_Appears in:_
- [OrderSpec](#orderspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `srcRef` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#localobjectreference-v1-core)_ | SrcRef defines which Endpoint object is used as fallback source by all artifacts. |  |  |
| `dstRef` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#localobjectreference-v1-core)_ | DstRef defines which Endpoint object is used as fallback destination by all artifacts. |  |  |
| `cron` _[Cron](#cron)_ | Cron specifies options which determine when the order should be scheduled. |  |  |




#### OrderSpec



OrderSpec defines the desired state of Order



_Appears in:_
- [Order](#order)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `defaults` _[OrderDefaults](#orderdefaults)_ | Defaults sets up defaults for all artifacts. |  |  |
| `artifacts` _[OrderArtifact](#orderartifact) array_ | Artifacts lists all artifacts, that will be processed by this Order. |  |  |
| `TTLSecondsAfterCompletion` _integer_ | TTLSecondsAfterCompletion specifies the time to live for the created ArtifactWorkflow(s) after completion.<br />After this time, the ArtifactWorkflow(s) are automatically deleted.<br />If unset, the ArtifactWorkflow(s) are automatically deleted immediately after completion. |  |  |


#### OrderStatus



OrderStatus defines the observed state of Order



_Appears in:_
- [Order](#order)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `artifactWorkflows` _object (keys:string, values:[OrderArtifactWorkflowStatus](#orderartifactworkflowstatus))_ | ArtifactWorkflows tracks the created workflows |  |  |
| `message` _string_ | A human readable message describing the current condition of the order. |  |  |
| `lastReconcileAt` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#time-v1-meta)_ | LastReconcileAt is the last time the Order was reconciled |  |  |
| `lastForceAt` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#time-v1-meta)_ | LastForceAt is the last time a force reconciliation was requested |  |  |


#### WorkflowPhase

_Underlying type:_ _string_

WorkflowPhase is an enum tracking in which phase a Workflow can be.



_Appears in:_
- [ArtifactWorkflowStatus](#artifactworkflowstatus)
- [OrderArtifactWorkflowStatus](#orderartifactworkflowstatus)
- [WorkflowStatus](#workflowstatus)

| Field | Description |
| --- | --- |
| `` |  |
| `Pending` |  |
| `Running` |  |
| `Succeeded` |  |
| `Failed` |  |
| `Error` |  |
| `Active` |  |
| `Stopped` |  |


#### WorkflowStatus







_Appears in:_
- [ArtifactWorkflowStatus](#artifactworkflowstatus)
- [OrderArtifactWorkflowStatus](#orderartifactworkflowstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `phase` _[WorkflowPhase](#workflowphase)_ | Phase tracks which phase the corresponding Workflow is in |  |  |
| `message` _string_ | A human readable message describing the current condition of the artifact workflow. |  |  |
| `completionTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#time-v1-meta)_ | CompletionTime is the time when the workflow finished |  |  |
| `lastScheduled` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#time-v1-meta)_ | LastScheduled is the last time the workflow was scheduled via cron |  |  |
| `succeeded` _integer_ | Succeeded counts how many times child workflows succeeded |  |  |
| `failed` _integer_ | Failed counts how many times child workflows failed |  |  |


