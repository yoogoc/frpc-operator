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

	"github.com/YoogoC/frpc-operator/builder"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	frpcv1 "github.com/YoogoC/frpc-operator/api/v1"
)

// ClientReconciler reconciles a Client object
type ClientReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const myFinalizerName = "frpc.yoogo.top/finalizer"

// +kubebuilder:rbac:groups=frpc.yoogo.top,resources=clients,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=frpc.yoogo.top,resources=clients/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=frpc.yoogo.top,resources=clients/finalizers,verbs=update

// +kubebuilder:rbac:groups="apps",resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=roles,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Client object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *ClientReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	log.Log.Info(req.Name)
	frpClient := new(frpcv1.Client)
	if err := r.Client.Get(ctx, req.NamespacedName, frpClient); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if frpClient.DeletionTimestamp == nil {
		if !controllerutil.ContainsFinalizer(frpClient, myFinalizerName) {
			controllerutil.AddFinalizer(frpClient, myFinalizerName)
			if err := r.Update(ctx, frpClient); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(frpClient, myFinalizerName) {
			// our finalizer is present, so lets handle any external dependency
			if err := r.deleteExternalResources(ctx, req.NamespacedName); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return ctrl.Result{}, err
			}
			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(frpClient, myFinalizerName)
			if err := r.Update(ctx, frpClient); err != nil {
				return ctrl.Result{}, err
			}
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}
	// 2. 如果不是删除,根据client和proxy的定义生成frpc.ini
	if err := createOrUpdateConfigMap(ctx, r.Client, frpClient); err != nil {
		return ctrl.Result{}, err
	}

	serviceAccountName := "frpc-config-reload"
	roleName := "frpc-config-reload"
	roleBindingName := "frpc-config-reload-binding"
	if err := tryCreateRbac(ctx, r.Client, req.Namespace, serviceAccountName, roleName, roleBindingName); err != nil {
		return ctrl.Result{}, err
	}

	// 4. 尝试找到同名的deploy,找不到就创建,找到就更新
	deploy := builder.NewDeployBuilder().
		SetName(req.Name).
		SetImage("fatedier/frpc:v0.44.0"). // TODO
		SetNamespace(req.Namespace).
		Build()

	oldDeploy := new(appsv1.Deployment)
	if err := r.Client.Get(ctx, req.NamespacedName, oldDeploy); err != nil {
		if apierrors.IsNotFound(err) {
			if err := r.Client.Create(ctx, deploy); err != nil {
				return ctrl.Result{}, err
			} else {
				return ctrl.Result{}, nil
			}
		}
		return ctrl.Result{}, err
	} else {
		err := r.Client.Update(ctx, deploy)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClientReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&frpcv1.Client{}).
		Complete(r)
}

func (r *ClientReconciler) deleteExternalResources(ctx context.Context, nn types.NamespacedName) error {
	err := r.Client.Delete(ctx, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nn.Name,
			Namespace: nn.Namespace,
		},
	})
	if err != nil {
		return err
	}
	err = r.Client.Delete(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nn.Name,
			Namespace: nn.Namespace,
		},
	})
	if err != nil {
		return err
	}
	// 获取当前命名空间下所有client,如果全部删除了,则删除rbac
	return nil
}
