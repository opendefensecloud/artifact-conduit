// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package orderinitializer_test

import (
	"context"
	"testing"
	"time"

	"go.opendefense.cloud/arc/client-go/clientset/versioned/fake"
	informers "go.opendefense.cloud/arc/client-go/informers/externalversions"
	"go.opendefense.cloud/arc/pkg/admission/orderinitializer"
	"k8s.io/apiserver/pkg/admission"
)

// TestWantsInternalOrderInformerFactory ensures that the informer factory is injected
// when the WantsInternalOrderInformerFactory interface is implemented by a plugin.
func TestWantsInternalOrderInformerFactory(t *testing.T) {
	cs := &fake.Clientset{}
	sf := informers.NewSharedInformerFactory(cs, time.Duration(1)*time.Second)
	target := orderinitializer.New(sf)

	wantOrderInformerFactory := &wantInternalOrderInformerFactory{}
	target.Initialize(wantOrderInformerFactory)
	if wantOrderInformerFactory.sf != sf {
		t.Errorf("expected informer factory to be initialized")
	}
}

// wantInternalOrderInformerFactory is a test stub that fulfills the WantsInternalOrderInformerFactory interface
type wantInternalOrderInformerFactory struct {
	sf informers.SharedInformerFactory
}

func (f *wantInternalOrderInformerFactory) SetInternalOrderInformerFactory(sf informers.SharedInformerFactory) {
	f.sf = sf
}
func (f *wantInternalOrderInformerFactory) Admit(ctx context.Context, a admission.Attributes, o admission.ObjectInterfaces) error {
	return nil
}
func (f *wantInternalOrderInformerFactory) Handles(o admission.Operation) bool { return false }
func (f *wantInternalOrderInformerFactory) ValidateInitialization() error      { return nil }

var _ admission.Interface = &wantInternalOrderInformerFactory{}
var _ orderinitializer.WantsInternalOrderInformerFactory = &wantInternalOrderInformerFactory{}
