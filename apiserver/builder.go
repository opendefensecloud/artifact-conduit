// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package apiserver

import (
	"fmt"
	"net"

	"github.com/spf13/cobra"
	"go.opendefense.cloud/arc/apiserver/rest"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/version"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/endpoints/openapi"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	"k8s.io/apiserver/pkg/util/compatibility"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/component-base/cli"
	basecompatibility "k8s.io/component-base/compatibility"
	"k8s.io/component-base/featuregate"
	baseversion "k8s.io/component-base/version"
	openapicommon "k8s.io/kube-openapi/pkg/common"
	netutils "k8s.io/utils/net"
)

// ExtraAdmissionInitializers is a callback that returns a SharedInformerFactory and admission plugin initializers.
type ExtraAdmissionInitializers func(*genericapiserver.RecommendedConfig) (SharedInformerFactory, []admission.PluginInitializer, error)

// RecommendedConfigFn is a callback that modifies the RecommendedConfig before the server starts.
type RecommendedConfigFn func(*genericapiserver.RecommendedConfig)

// SharedInformerFactory is used to start informer watching for resource changes.
type SharedInformerFactory interface {
	// Start begins watching resources and blocks until stopCh is closed.
	Start(stopCh <-chan struct{})
}

// APIGroupFn returns an APIGroupInfo for installing an API group into the server.
type APIGroupFn func(scheme *runtime.Scheme, codecs serializer.CodecFactory, c *genericapiserver.CompletedConfig) genericapiserver.APIGroupInfo

// Builder constructs and runs a Kubernetes API server with custom resource groups.
// It handles schema registration, storage configuration, admission, and lifecycle hooks.
type Builder struct {
	componentName                          string
	alternateDNS                           []string
	scheme                                 *runtime.Scheme
	codecs                                 serializer.CodecFactory
	groupVersions                          []schema.GroupVersion
	skipDefaultComponentGlobalsRegistrySet bool
	extraAdmissionInitializers             ExtraAdmissionInitializers
	sharedInformerFactories                []SharedInformerFactory
	recommendedOptions                     *genericoptions.RecommendedOptions
	componentGlobalsRegistry               basecompatibility.ComponentGlobalsRegistry
	recommendedConfigFns                   []RecommendedConfigFn
	apiGroupFns                            []APIGroupFn
}

// NewBuilder creates a new API server builder with the given runtime scheme.
func NewBuilder(scheme *runtime.Scheme) *Builder {
	return &Builder{
		scheme:                  scheme,
		codecs:                  serializer.NewCodecFactory(scheme),
		sharedInformerFactories: []SharedInformerFactory{},
		apiGroupFns:             []APIGroupFn{},
		groupVersions:           []schema.GroupVersion{},
	}
}

// WithComponentName sets the component name used for server identification and logging.
func (b *Builder) WithComponentName(n string) *Builder {
	b.componentName = n
	return b
}

// WithOpenAPIDefinitions configures OpenAPI (Swagger) documentation for the API server.
func (b *Builder) WithOpenAPIDefinitions(name, version string, defs openapicommon.GetOpenAPIDefinitions) *Builder {
	b.recommendedConfigFns = append(b.recommendedConfigFns, func(config *genericapiserver.RecommendedConfig) {
		config.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(defs, openapi.NewDefinitionNamer(b.scheme))
		config.OpenAPIConfig.Info.Title = name
		config.OpenAPIConfig.Info.Version = version

		config.OpenAPIV3Config = genericapiserver.DefaultOpenAPIV3Config(defs, openapi.NewDefinitionNamer(b.scheme))
		config.OpenAPIV3Config.Info.Title = name
		config.OpenAPIV3Config.Info.Version = name
	})
	return b
}

// WithAPIGroupFn registers an APIGroupFn to install an API group into the server.
func (b *Builder) WithAPIGroupFn(fn APIGroupFn) *Builder {
	if fn == nil {
		return b
	}
	b.apiGroupFns = append(b.apiGroupFns, fn)
	return b
}

// With registers a ResourceHandler's API group and group versions.
func (b *Builder) With(rh ResourceHandler) *Builder {
	_ = b.WithAPIGroupFn(rh.apiGroupFn)
	return b.WithGroupVersions(rh.groupVersions...)
}

// WithExtraAdmissionInitializers sets custom admission plugin initialization logic.
func (b *Builder) WithExtraAdmissionInitializers(f ExtraAdmissionInitializers) *Builder {
	if f == nil {
		return b
	}
	b.extraAdmissionInitializers = f
	return b
}

// WithSharedInformerFactory registers a SharedInformerFactory to be started when the server starts.
func (b *Builder) WithSharedInformerFactory(f SharedInformerFactory) *Builder {
	if f == nil {
		return b
	}
	b.sharedInformerFactories = append(b.sharedInformerFactories, f)
	return b
}

// WithGroupVersions appends the  group versions to configure storage
// encoding/decoding for the API server. This must be provided by callers
// so that the storage codec matches the registered types in the scheme.
func (b *Builder) WithGroupVersions(gvs ...schema.GroupVersion) *Builder {
	b.groupVersions = append(b.groupVersions, gvs...)
	return b
}

// Execute builds and runs the API server, returning an exit code suitable for os.Exit().
// It configures storage, admission, informers, and launches the server with all registered resources.
func (b *Builder) Execute() int {
	// Validate that all group versions belong to the same API group.
	groupName := ""
	for _, gv := range b.groupVersions {
		if groupName != "" && groupName != gv.Group {
			panic("all exposed resources expected to have the same group")
		}
		groupName = gv.Group
	}
	// Get the ordered group versions to ensure storage encoding matches the registered types.
	orderedGroupVersions := b.scheme.PrioritizedVersionsForGroup(groupName)

	// Set up default recommended options if not already configured.
	if b.recommendedOptions == nil {
		b.recommendedOptions = genericoptions.NewRecommendedOptions(
			fmt.Sprintf("/registry/%s", groupName),
			b.codecs.LegacyCodec(orderedGroupVersions...),
		)
	}
	// Configure storage to use the ordered group versions for encoding.
	b.recommendedOptions.Etcd.StorageConfig.EncodeVersioner = schema.GroupVersions(orderedGroupVersions)
	// Wire up admission initializers if provided.
	if b.extraAdmissionInitializers != nil {
		b.recommendedOptions.ExtraAdmissionInitializers = func(c *genericapiserver.RecommendedConfig) ([]admission.PluginInitializer, error) {
			informerFactory, pluginInitialisers, err := b.extraAdmissionInitializers(c)
			if err != nil {
				return nil, err
			}
			// Collect informer factories from admission setup.
			b.sharedInformerFactories = append(b.sharedInformerFactories, informerFactory)
			return pluginInitialisers, nil
		}
	}
	// Set up TLS certificates for secure serving.
	utilruntime.Must(b.recommendedOptions.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", b.alternateDNS, []net.IP{netutils.ParseIPSloppy("127.0.0.1")}))

	// Use default component registry if not provided.
	if b.componentGlobalsRegistry == nil {
		b.componentGlobalsRegistry = compatibility.DefaultComponentGlobalsRegistry
	}

	ctx := genericapiserver.SetupSignalContext()
	cmd := &cobra.Command{
		Short: "Launch API server",
		Long:  "Launch API server",
		PersistentPreRunE: func(*cobra.Command, []string) error {
			if b.skipDefaultComponentGlobalsRegistrySet {
				return nil
			}
			return b.componentGlobalsRegistry.Set()
		},
		RunE: func(c *cobra.Command, args []string) error {
			// Validate essential builder configuration early to provide a helpful error
			if len(orderedGroupVersions) == 0 {
				return fmt.Errorf("orderedGroupVersions not set on Builder; call WithGroupVersions(...) before Execute")
			}
			// Collect and validate all configuration.
			errors := []error{}
			errors = append(errors, b.recommendedOptions.Validate()...)
			errors = append(errors, b.componentGlobalsRegistry.Validate()...)
			if err := utilerrors.NewAggregate(errors); err != nil {
				return err
			}

			serverConfig := genericapiserver.NewRecommendedConfig(b.codecs)

			// Apply custom configuration functions.
			for _, fn := range b.recommendedConfigFns {
				fn(serverConfig)
			}

			// Set feature gates and versioning.
			serverConfig.FeatureGate = b.componentGlobalsRegistry.FeatureGateFor(basecompatibility.DefaultKubeComponent)
			serverConfig.EffectiveVersion = b.componentGlobalsRegistry.EffectiveVersionFor(b.componentName)

			// Apply recommended options (TLS, etcd, admission, etc.).
			if err := b.recommendedOptions.ApplyTo(serverConfig); err != nil {
				return err
			}

			// Create the fully configured API server.
			completedConfig := serverConfig.Complete()
			server, err := completedConfig.New(fmt.Sprintf("%s-apiserver", b.componentName), genericapiserver.NewEmptyDelegate())
			if err != nil {
				return err
			}

			// Build API groups from registered handlers and install them into the server.
			apiGroupMap := map[string]*genericapiserver.APIGroupInfo{}
			for _, fn := range b.apiGroupFns {
				apiGroupInfo := fn(b.scheme, b.codecs, &completedConfig)
				groupName := ""
				for _, gv := range apiGroupInfo.PrioritizedVersions {
					groupName = gv.Group
					break
				}
				if groupName == "" {
					return fmt.Errorf("empty group name is not allowed")
				}

				// Merge resources from multiple handlers for the same group.
				if apiGroupInfoPrev, ok := apiGroupMap[groupName]; ok {
					apiGroupInfoPrev.VersionedResourcesStorageMap = mergeVersionedResourcesStorageMap(apiGroupInfoPrev.VersionedResourcesStorageMap, apiGroupInfo.VersionedResourcesStorageMap)
				} else {
					apiGroupMap[groupName] = &apiGroupInfo
				}

			}

			// Install all API groups into the server.
			for _, apiGroupInfo := range apiGroupMap {
				if err := server.InstallAPIGroup(apiGroupInfo); err != nil {
					return err
				}
			}

			// Register post-start hook to start informers once server is ready.
			server.AddPostStartHookOrDie(fmt.Sprintf("start-%s-server-informers", b.componentName), func(context genericapiserver.PostStartHookContext) error {
				// Defensive: the SharedInformerFactory may not be set by the recommended options
				// in all call sites (callers may provide their own factories via WithSharedInformerFactory).
				// Avoid a nil-pointer panic by checking for nil before starting.
				if serverConfig.SharedInformerFactory != nil {
					serverConfig.SharedInformerFactory.Start(context.Done())
				}
				for _, sharedInformerFactory := range b.sharedInformerFactories {
					sharedInformerFactory.Start(context.Done())
				}
				return nil
			})

			return server.PrepareRun().RunWithContext(ctx)
		},
	}
	cmd.SetContext(ctx)

	flags := cmd.Flags()
	b.recommendedOptions.AddFlags(flags)

	// Register component versions and feature gates with the global registry.
	// TODO: expose to builder
	defaultVersion := "1.2"
	// Register the "ARC" component with the global component registry,
	// associating it with its effective version and feature gate configuration.
	// Will skip if the component has been registered, like in the integration test.
	_, _ = b.componentGlobalsRegistry.ComponentGlobalsOrRegister(
		b.componentName, basecompatibility.NewEffectiveVersionFromString(defaultVersion, "", ""),
		featuregate.NewVersionedFeatureGate(version.MustParse(defaultVersion)))

	// Add versioned feature specifications for the "BanFlunder" feature.
	// These specifications, together with the effective version, determine if the feature is enabled.
	// TODO: expose to builder
	// utilruntime.Must(arcFeatureGate.AddVersioned(map[featuregate.Feature]featuregate.VersionedSpecs{
	// 	"BanFlunder": {
	// 		{Version: version.MustParse("1.0"), Default: false, PreRelease: featuregate.Alpha},
	// 		{Version: version.MustParse("1.1"), Default: true, PreRelease: featuregate.Beta},
	// 		{Version: version.MustParse("1.2"), Default: true, PreRelease: featuregate.GA, LockToDefault: true},
	// 	},
	// }))

	// Register the default kube component if not already present in the global registry.
	_, _ = b.componentGlobalsRegistry.ComponentGlobalsOrRegister(basecompatibility.DefaultKubeComponent,
		basecompatibility.NewEffectiveVersionFromString(baseversion.DefaultKubeBinaryVersion, "", ""), utilfeature.DefaultMutableFeatureGate)

	// Set the emulation version mapping from the "ARC" component to the kube component.
	// This ensures that the emulation version of the latter is determined by the emulation version of the former.

	versionToKubeVersion := func(ver *version.Version) *version.Version {
		if ver.Major() != 1 {
			return nil
		}
		kubeVer := version.MustParse(baseversion.DefaultKubeBinaryVersion)
		// "1.2" maps to kubeVer
		offset := int(ver.Minor()) - 2
		mappedVer := kubeVer.OffsetMinor(offset)
		if mappedVer.GreaterThan(kubeVer) {
			return kubeVer
		}
		return mappedVer
	}
	utilruntime.Must(b.componentGlobalsRegistry.SetEmulationVersionMapping(b.componentName, basecompatibility.DefaultKubeComponent, versionToKubeVersion))

	b.componentGlobalsRegistry.AddFlags(flags)

	// TODO: add kube version compatibility matrix and feature gates

	return cli.Run(cmd)
}

// mergeVersionedResourcesStorageMap combines two versioned storage maps, allowing multiple
// handlers to contribute resources to the same API group version.
func mergeVersionedResourcesStorageMap(a map[string]map[string]rest.Storage, b map[string]map[string]rest.Storage) map[string]map[string]rest.Storage {
	c := map[string]map[string]rest.Storage{}
	// Copy all entries from a into c.
	for version, storeMap := range a {
		if _, ok := c[version]; !ok {
			c[version] = map[string]rest.Storage{}
		}
		for resource, store := range storeMap {
			c[version][resource] = store
		}
	}
	// Merge entries from b into c.
	for version, storeMap := range b {
		if _, ok := c[version]; !ok {
			c[version] = map[string]rest.Storage{}
		}
		for resource, store := range storeMap {
			c[version][resource] = store
		}
	}
	return c
}
