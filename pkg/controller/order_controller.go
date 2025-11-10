// Copyright 2025 BWI GmbH and Artefact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	arcv1alpha1 "gitlab.opencode.de/bwi/ace/artifact-conduit/api/arc/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// OrderReconciler reconciles a Order object
type OrderReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=arc.bwi.de,resources=fragments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=arc.bwi.de,resources=orders,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=arc.bwi.de,resources=orders/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=arc.bwi.de,resources=orders/finalizers,verbs=update

// Reconcile moves the current state of the cluster closer to the desired state
func (r *OrderReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Fetch the Order instance
	order := &arcv1alpha1.Order{}
	if err := r.Get(ctx, req.NamespacedName, order); err != nil {
		if apierrors.IsNotFound(err) {
			// Object not found, return. Created objects are automatically garbage collected.
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	desiredFrags := map[string]*arcv1alpha1.Fragment{}
	for _, artifact := range order.Spec.Artifacts {
		// Let's collect the necessary data for the fragment from the artifact and order
		frag := &arcv1alpha1.Fragment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: order.Namespace,
				Name:      "",
			},
			Spec: arcv1alpha1.FragmentSpec{
				Type:   artifact.Type,
				SrcRef: artifact.SrcRef,
				DstRef: artifact.DstRef,
				Spec:   artifact.Spec,
			},
		}
		if frag.Spec.SrcRef.Name == "" {
			frag.Spec.SrcRef = order.Spec.Defaults.SrcRef
		}
		if frag.Spec.DstRef.Name == "" {
			frag.Spec.DstRef = order.Spec.Defaults.DstRef
		}

		// Create a hash based on fragment fields for idempotency and compute the fragment name
		h := sha256.New()
		data := map[string]interface{}{
			"type": frag.Spec.Type,
			"src":  frag.Spec.SrcRef.Name,
			"dst":  frag.Spec.DstRef.Name,
			"spec": frag.Spec.Spec.Raw,
		}
		jsonData, err := json.Marshal(data)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to marshal fragment data: %w", err)
		}
		h.Write(jsonData)
		sha := hex.EncodeToString(h.Sum(nil))[:16]
		frag.Name = fmt.Sprintf("%s-%s", order.Name, sha)

		desiredFrags[sha] = frag
	}

	// List missing fragments
	fragsToCreate := []string{}
	for sha, _ := range desiredFrags {
		_, exists := order.Status.Fragments[sha]
		if exists {
			continue
		}
		fragsToCreate = append(fragsToCreate, sha)
	}

	// Find obsolete fragments
	fragsToDelete := []string{}
	for sha, _ := range order.Status.Fragments {
		_, exists := desiredFrags[sha]
		if exists {
			continue
		}
		fragsToDelete = append(fragsToDelete, sha)
	}

	// Create missing fragments
	for _, sha := range fragsToCreate {
		frag := desiredFrags[sha]

		// Set owner reference so Fragment is garbage-collected with the Order
		if err := controllerutil.SetControllerReference(order, frag, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}

		if err := r.Create(ctx, frag); err != nil {
			if apierrors.IsAlreadyExists(err) {
				// Already created by a previous reconcile — that's fine
				continue
			}
			return ctrl.Result{}, err
		}

		// Update status
		order.Status.Fragments[sha] = corev1.LocalObjectReference{Name: frag.Name}
	}

	// Delete obsolete fragments
	for _, sha := range fragsToDelete {
		// Does not exist anymore, let's clean up!
		if err := r.Delete(ctx, &arcv1alpha1.Fragment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: order.Namespace,
				Name:      order.Status.Fragments[sha].Name,
			},
		}); err != nil {
			return ctrl.Result{}, err
		}

		// Update status
		delete(order.Status.Fragments, sha)
	}

	// Update status
	if err := r.Status().Update(ctx, order); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OrderReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arcv1alpha1.Order{}).
		Owns(&arcv1alpha1.Fragment{}).
		Complete(r)
}
