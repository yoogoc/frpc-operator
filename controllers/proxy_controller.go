/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	frpcv1 "github.com/YoogoC/frpc-operator/api/v1"
)

// ProxyReconciler reconciles a Proxy object
type ProxyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=frpc.yoogo.top,resources=proxies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=frpc.yoogo.top,resources=proxies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=frpc.yoogo.top,resources=proxies/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Proxy object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *ProxyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	log.Log.Info(req.Name)
	proxy := new(frpcv1.Proxy)
	if err := r.Get(ctx, req.NamespacedName, proxy); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	if proxy.DeletionTimestamp == nil {
		if !controllerutil.ContainsFinalizer(proxy, myFinalizerName) {
			controllerutil.AddFinalizer(proxy, myFinalizerName)
			if err := r.Update(ctx, proxy); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(proxy, myFinalizerName) {
			// our finalizer is present, so lets handle any external dependency
			if err := r.reloadConfigMap(ctx, proxy, req.NamespacedName); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return ctrl.Result{}, err
			}
			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(proxy, myFinalizerName)
			if err := r.Update(ctx, proxy); err != nil {
				return ctrl.Result{}, err
			}
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, r.reloadConfigMap(ctx, proxy, req.NamespacedName)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProxyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&frpcv1.Proxy{}).
		Complete(r)
}

func (r *ProxyReconciler) reloadConfigMap(ctx context.Context, proxy *frpcv1.Proxy, nn types.NamespacedName) error {
	frpClient := new(frpcv1.Client)
	if err := r.Get(ctx, client.ObjectKey{Name: proxy.Spec.Client, Namespace: nn.Namespace}, frpClient); err != nil {
		return err
	}
	if frpClient.DeletionTimestamp != nil {
		return nil
	}
	if err := createOrUpdateConfigMap(ctx, r.Client, frpClient); err != nil {
		return err
	}
	return nil
}
