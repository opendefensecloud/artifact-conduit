// Copyright 2025 BWI GmbH and Artefact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"

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

	// Create a Fragment owned by this Order
	frag := &arcv1alpha1.Fragment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    order.Namespace,
			GenerateName: order.Name + "-fragment-",
		},
		Spec: arcv1alpha1.FragmentSpec{
			// Intentionally empty defaults; controllers may fill this in later.
			// Provide a minimal SrcRef/DstRef if needed:
			SrcRef: corev1.LocalObjectReference{},
			DstRef: corev1.LocalObjectReference{},
		},
	}

	// Set owner reference so Fragment is garbage-collected with the Order
	if err := controllerutil.SetControllerReference(order, frag, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.Create(ctx, frag); err != nil {
		if apierrors.IsAlreadyExists(err) {
			// Already created by a previous reconcile — that's fine
			return ctrl.Result{}, nil
		}
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
