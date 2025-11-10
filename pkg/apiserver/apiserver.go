// Copyright 2025 BWI GmbH and Artefact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package apiserver

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"

	"gitlab.opencode.de/bwi/ace/artifact-conduit/api/arc"
	"gitlab.opencode.de/bwi/ace/artifact-conduit/api/arc/install"
	arcregistry "gitlab.opencode.de/bwi/ace/artifact-conduit/pkg/registry"
	fragmentstorage "gitlab.opencode.de/bwi/ace/artifact-conduit/pkg/registry/fragment"
	orderstorage "gitlab.opencode.de/bwi/ace/artifact-conduit/pkg/registry/order"
)

var (
	// Scheme defines methods for serializing and deserializing API objects.
	Scheme = runtime.NewScheme()
	// Codecs provides methods for retrieving codecs and serializers for specific
	// versions and content types.
	Codecs           = serializer.NewCodecFactory(Scheme)
	ARCComponentName = "arc"
)

func init() {
	install.Install(Scheme)

	// we need to add the options to empty v1
	// TODO: fix the server code to avoid this
	metav1.AddToGroupVersion(Scheme, schema.GroupVersion{Version: "v1"})

	// TODO: keep the generic API server from wanting this
	unversioned := schema.GroupVersion{Group: "", Version: "v1"}
	Scheme.AddUnversionedTypes(unversioned,
		&metav1.Status{},
		&metav1.APIVersions{},
		&metav1.APIGroupList{},
		&metav1.APIGroup{},
		&metav1.APIResourceList{},
	)
}

// ExtraConfig holds custom apiserver config
type ExtraConfig struct {
	// Place you custom config here.
}

// Config defines the config for the apiserver
type Config struct {
	GenericConfig *genericapiserver.RecommendedConfig
	ExtraConfig   ExtraConfig
}

// ARCServer contains state for a Kubernetes cluster master/api server.
type ARCServer struct {
	GenericAPIServer *genericapiserver.GenericAPIServer
}

type completedConfig struct {
	GenericConfig genericapiserver.CompletedConfig
	ExtraConfig   *ExtraConfig
}

// CompletedConfig embeds a private pointer that cannot be instantiated outside of this package.
type CompletedConfig struct {
	*completedConfig
}

// Complete fills in any fields not set that are required to have valid data. It's mutating the receiver.
func (cfg *Config) Complete() CompletedConfig {
	c := completedConfig{
		cfg.GenericConfig.Complete(),
		&cfg.ExtraConfig,
	}

	return CompletedConfig{&c}
}

// New returns a new instance of ARCServer from the given config.
func (c completedConfig) New() (*ARCServer, error) {
	genericServer, err := c.GenericConfig.New("arc-apiserver", genericapiserver.NewEmptyDelegate())
	if err != nil {
		return nil, err
	}

	s := &ARCServer{
		GenericAPIServer: genericServer,
	}

	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(arc.GroupName, Scheme, metav1.ParameterCodec, Codecs)

	v1alpha1storage := map[string]rest.Storage{}
	v1alpha1storage["orders"] = arcregistry.RESTInPeace(orderstorage.NewREST(Scheme, c.GenericConfig.RESTOptionsGetter))
	v1alpha1storage["fragments"] = arcregistry.RESTInPeace(fragmentstorage.NewREST(Scheme, c.GenericConfig.RESTOptionsGetter))
	apiGroupInfo.VersionedResourcesStorageMap["v1alpha1"] = v1alpha1storage

	// v1beta1storage := map[string]rest.Storage{}
	// v1beta1storage["orders"] = arcregistry.RESTInPeace(orderstorage.NewREST(Scheme, c.GenericConfig.RESTOptionsGetter))
	// apiGroupInfo.VersionedResourcesStorageMap["v1beta1"] = v1beta1storage

	if err := s.GenericAPIServer.InstallAPIGroup(&apiGroupInfo); err != nil {
		return nil, err
	}

	return s, nil
}
