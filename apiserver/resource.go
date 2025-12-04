package apiserver

import (
	"context"

	"go.opendefense.cloud/arc/apiserver/resource"
	"go.opendefense.cloud/arc/apiserver/rest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/server"
)

type ResourceHandler struct {
	groupVersions []schema.GroupVersion
	apiGroupFn    APIGroupFn
}

func Resource[E resource.Object, T resource.ObjectWithDeepCopy[E]](obj T, gvs ...schema.GroupVersion) ResourceHandler {
	return ResourceHandler{
		groupVersions: gvs,
		apiGroupFn: func(scheme *runtime.Scheme, codecs serializer.CodecFactory, c *server.CompletedConfig) server.APIGroupInfo {
			gr := obj.GetGroupResource()
			strategy := rest.NewDefaultStrategy(obj, scheme, gr)
			store, err := rest.NewStore(scheme, obj.New, obj.NewList, gr, strategy, c.RESTOptionsGetter)
			if err != nil {
				panic(err)
			}

			storage := map[string]rest.Storage{}
			storage[gr.Resource] = store

			if _, ok := any(obj).(resource.ObjectWithStatusSubResource); ok {
				statusPrepareForUpdate := func(ctx context.Context, obj, old runtime.Object) {
					// We copy status to old
					statusObj := any(obj).(resource.ObjectWithStatusSubResource)
					statusObj.CopyStatusTo(old)
					// And use old (with new status) to reset spec of new obj
					copyableObj := any(obj).(E)
					copyableOld := any(old).(T)
					copyableOld.DeepCopyInto(copyableObj)
				}
				statusStore := *store
				statusStore.UpdateStrategy = &rest.PrepareForUpdaterStrategy{
					RESTUpdateStrategy: store.UpdateStrategy,
					OverrideFn:         statusPrepareForUpdate,
				}
				storage[gr.Resource+"/status"] = &statusStore
			}

			apiGroupInfo := server.NewDefaultAPIGroupInfo(gr.Group, scheme, metav1.ParameterCodec, codecs)

			for _, gv := range gvs {
				if gv.Group != gr.Group {
					panic("unexpected group mismatch")
				}
				apiGroupInfo.VersionedResourcesStorageMap[gv.Version] = storage
			}

			return apiGroupInfo
		},
	}
}
