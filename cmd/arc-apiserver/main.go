// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"

	"go.opendefense.cloud/arc/api/arc"
	"go.opendefense.cloud/arc/api/arc/install"
	arcv1alpha1 "go.opendefense.cloud/arc/api/arc/v1alpha1"
	"go.opendefense.cloud/arc/apiserver"
	clientset "go.opendefense.cloud/arc/client-go/clientset/versioned"
	informers "go.opendefense.cloud/arc/client-go/informers/externalversions"
	"go.opendefense.cloud/arc/client-go/openapi"
	"go.opendefense.cloud/arc/pkg/admission/orderinitializer"
	arcregistry "go.opendefense.cloud/arc/pkg/registry"
	artifacttypestorage "go.opendefense.cloud/arc/pkg/registry/artifacttype"
	artifactworkflowstorage "go.opendefense.cloud/arc/pkg/registry/artifactworkflow"
	clusterartifacttypestorage "go.opendefense.cloud/arc/pkg/registry/clusterartifacttype"
	endpointstorage "go.opendefense.cloud/arc/pkg/registry/endpoint"
	orderstorage "go.opendefense.cloud/arc/pkg/registry/order"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/server"
)

const (
	componentName = "arc"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	install.Install(scheme)

	// we need to add the options to empty v1
	// TODO: fix the server code to avoid this
	metav1.AddToGroupVersion(scheme, schema.GroupVersion{Version: "v1"})

	// TODO: keep the generic API server from wanting this
	unversioned := schema.GroupVersion{Group: "", Version: "v1"}
	scheme.AddUnversionedTypes(unversioned,
		&metav1.Status{},
		&metav1.APIVersions{},
		&metav1.APIGroupList{},
		&metav1.APIGroup{},
		&metav1.APIResourceList{},
	)
}
func main() {
	code := apiserver.NewBuilder(scheme).
		WithComponentName(componentName).
		WithGroupName(arc.GroupName).
		WithOpenAPIDefinitions(componentName, "v0.1.0", openapi.GetOpenAPIDefinitions).
		WithExtraAdmissionInitializers(func(c *server.RecommendedConfig) (apiserver.SharedInformerFactory, []admission.PluginInitializer, error) {
			client, err := clientset.NewForConfig(c.LoopbackClientConfig)
			if err != nil {
				return nil, nil, err
			}
			informerFactory := informers.NewSharedInformerFactory(client, c.LoopbackClientConfig.Timeout)
			return informerFactory, []admission.PluginInitializer{orderinitializer.New(informerFactory)}, nil
		}).
		// TODO: refactor how we construct api groups and storage registries
		WithAPIGroupFn(func(scheme *runtime.Scheme, codecs serializer.CodecFactory, c *server.CompletedConfig) server.APIGroupInfo {
			apiGroupInfo := server.NewDefaultAPIGroupInfo(arc.GroupName, scheme, metav1.ParameterCodec, codecs)

			v1alpha1storage := map[string]rest.Storage{}
			v1alpha1storage["orders"] = arcregistry.RESTInPeace(orderstorage.NewREST(scheme, c.RESTOptionsGetter))
			v1alpha1storage["orders/status"] = arcregistry.RESTInPeace(orderstorage.NewStatusREST(scheme, c.RESTOptionsGetter))
			v1alpha1storage["artifactworkflows"] = arcregistry.RESTInPeace(artifactworkflowstorage.NewREST(scheme, c.RESTOptionsGetter))
			v1alpha1storage["artifactworkflows/status"] = arcregistry.RESTInPeace(artifactworkflowstorage.NewStatusREST(scheme, c.RESTOptionsGetter))
			v1alpha1storage["endpoints"] = arcregistry.RESTInPeace(endpointstorage.NewREST(scheme, c.RESTOptionsGetter))
			v1alpha1storage["artifacttypes"] = arcregistry.RESTInPeace(artifacttypestorage.NewREST(scheme, c.RESTOptionsGetter))
			v1alpha1storage["clusterartifacttypes"] = arcregistry.RESTInPeace(clusterartifacttypestorage.NewREST(scheme, c.RESTOptionsGetter))
			apiGroupInfo.VersionedResourcesStorageMap["v1alpha1"] = v1alpha1storage

			return apiGroupInfo
		}).
		WithOrderedGroupVersions([]schema.GroupVersion{
			arcv1alpha1.SchemeGroupVersion,
		}).
		Execute()
	os.Exit(code)
}
