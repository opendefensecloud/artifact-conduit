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

type ExtraAdmissionInitializers func(*genericapiserver.RecommendedConfig) (SharedInformerFactory, []admission.PluginInitializer, error)
type RecommendedConfigFn func(*genericapiserver.RecommendedConfig)

type SharedInformerFactory interface {
	Start(stopCh <-chan struct{})
}

type APIGroupFn func(scheme *runtime.Scheme, codecs serializer.CodecFactory, c *genericapiserver.CompletedConfig) genericapiserver.APIGroupInfo

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

func NewBuilder(scheme *runtime.Scheme) *Builder {
	return &Builder{
		scheme:                  scheme,
		codecs:                  serializer.NewCodecFactory(scheme),
		sharedInformerFactories: []SharedInformerFactory{},
		apiGroupFns:             []APIGroupFn{},
		groupVersions:           []schema.GroupVersion{},
	}
}

func (b *Builder) WithComponentName(n string) *Builder {
	b.componentName = n
	return b
}

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

func (b *Builder) WithAPIGroupFn(fn APIGroupFn) *Builder {
	if fn == nil {
		return b
	}
	b.apiGroupFns = append(b.apiGroupFns, fn)
	return b
}

func (b *Builder) With(rh ResourceHandler) *Builder {
	_ = b.WithAPIGroupFn(rh.apiGroupFn)
	return b.WithGroupVersions(rh.groupVersions...)
}

func (b *Builder) WithExtraAdmissionInitializers(f ExtraAdmissionInitializers) *Builder {
	if f == nil {
		return b
	}
	b.extraAdmissionInitializers = f
	return b
}

func (b *Builder) WithSharedInformerFactory(f SharedInformerFactory) *Builder {
	if f == nil {
		return b
	}
	b.sharedInformerFactories = append(b.sharedInformerFactories, f)
	return b
}

// WithGroupVersions sets the ordered group versions which are used
// to configure storage encoding/decoding for the API server. This must be
// provided by callers so that the storage codec matches the registered types
// in the scheme.
func (b *Builder) WithGroupVersions(gvs ...schema.GroupVersion) *Builder {
	b.groupVersions = append(b.groupVersions, gvs...)
	return b
}

func (b *Builder) Execute() int {
	groupName := ""
	for _, gv := range b.groupVersions {
		if groupName != "" && groupName != gv.Group {
			panic("all exposed resources expected to have the same group")
		}
		groupName = gv.Group
	}
	orderedGroupVersions := b.scheme.PrioritizedVersionsForGroup(groupName)

	if b.recommendedOptions == nil {
		b.recommendedOptions = genericoptions.NewRecommendedOptions(
			fmt.Sprintf("/registry/%s", groupName),
			b.codecs.LegacyCodec(orderedGroupVersions...),
		)
	}
	b.recommendedOptions.Etcd.StorageConfig.EncodeVersioner = schema.GroupVersions(orderedGroupVersions)
	if b.extraAdmissionInitializers != nil {
		b.recommendedOptions.ExtraAdmissionInitializers = func(c *genericapiserver.RecommendedConfig) ([]admission.PluginInitializer, error) {
			informerFactory, pluginInitialisers, err := b.extraAdmissionInitializers(c)
			if err != nil {
				return nil, err
			}
			b.sharedInformerFactories = append(b.sharedInformerFactories, informerFactory)
			return pluginInitialisers, nil
		}
	}
	utilruntime.Must(b.recommendedOptions.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", b.alternateDNS, []net.IP{netutils.ParseIPSloppy("127.0.0.1")}))

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
				return fmt.Errorf("orderedGroupVersions not set on Builder; call WithOrderedGroupVersions(...) before Execute")
			}
			errors := []error{}
			errors = append(errors, b.recommendedOptions.Validate()...)
			errors = append(errors, b.componentGlobalsRegistry.Validate()...)
			if err := utilerrors.NewAggregate(errors); err != nil {
				return err
			}

			serverConfig := genericapiserver.NewRecommendedConfig(b.codecs)

			for _, fn := range b.recommendedConfigFns {
				fn(serverConfig)
			}

			serverConfig.FeatureGate = b.componentGlobalsRegistry.FeatureGateFor(basecompatibility.DefaultKubeComponent)
			serverConfig.EffectiveVersion = b.componentGlobalsRegistry.EffectiveVersionFor(b.componentName)

			if err := b.recommendedOptions.ApplyTo(serverConfig); err != nil {
				return err
			}

			completedConfig := serverConfig.Complete()
			server, err := completedConfig.New(fmt.Sprintf("%s-apiserver", b.componentName), genericapiserver.NewEmptyDelegate())
			if err != nil {
				return err
			}

			// TODO: install API groups with their storage backends!
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

				if apiGroupInfoPrev, ok := apiGroupMap[groupName]; ok {
					apiGroupInfoPrev.VersionedResourcesStorageMap = mergeVersionedResourcesStorageMap(apiGroupInfoPrev.VersionedResourcesStorageMap, apiGroupInfo.VersionedResourcesStorageMap)
				} else {
					apiGroupMap[groupName] = &apiGroupInfo
				}

			}

			for _, apiGroupInfo := range apiGroupMap {
				if err := server.InstallAPIGroup(apiGroupInfo); err != nil {
					return err
				}
			}

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

func mergeVersionedResourcesStorageMap(a map[string]map[string]rest.Storage, b map[string]map[string]rest.Storage) map[string]map[string]rest.Storage {
	c := map[string]map[string]rest.Storage{}
	for version, storeMap := range a {
		if _, ok := c[version]; !ok {
			c[version] = map[string]rest.Storage{}
		}
		for resource, store := range storeMap {
			c[version][resource] = store
		}
	}
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
