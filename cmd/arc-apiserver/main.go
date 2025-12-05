// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"

	"go.opendefense.cloud/arc/api/arc"
	"go.opendefense.cloud/arc/api/arc/install"
	arcv1alpha1 "go.opendefense.cloud/arc/api/arc/v1alpha1"
	clientset "go.opendefense.cloud/arc/client-go/clientset/versioned"
	informers "go.opendefense.cloud/arc/client-go/informers/externalversions"
	"go.opendefense.cloud/arc/client-go/openapi"
	"go.opendefense.cloud/arc/pkg/admission/orderinitializer"
	"go.opendefense.cloud/sl/apiserver"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/admission"
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
		WithOpenAPIDefinitions(componentName, "v0.1.0", openapi.GetOpenAPIDefinitions).
		WithExtraAdmissionInitializers(func(c *server.RecommendedConfig) (apiserver.SharedInformerFactory, []admission.PluginInitializer, error) {
			client, err := clientset.NewForConfig(c.LoopbackClientConfig)
			if err != nil {
				return nil, nil, err
			}
			informerFactory := informers.NewSharedInformerFactory(client, c.LoopbackClientConfig.Timeout)
			return informerFactory, []admission.PluginInitializer{orderinitializer.New(informerFactory)}, nil
		}).
		With(apiserver.Resource(&arc.Order{}, arcv1alpha1.SchemeGroupVersion)).
		With(apiserver.Resource(&arc.ArtifactWorkflow{}, arcv1alpha1.SchemeGroupVersion)).
		With(apiserver.Resource(&arc.Endpoint{}, arcv1alpha1.SchemeGroupVersion)).
		With(apiserver.Resource(&arc.ArtifactType{}, arcv1alpha1.SchemeGroupVersion)).
		With(apiserver.Resource(&arc.ClusterArtifactType{}, arcv1alpha1.SchemeGroupVersion)).
		Execute()
	os.Exit(code)
}
