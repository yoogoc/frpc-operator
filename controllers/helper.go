package controllers

import (
	"context"

	frpcv1 "github.com/YoogoC/frpc-operator/api/v1"
	"github.com/YoogoC/frpc-operator/builder"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func createOrUpdateConfigMap(ctx context.Context, k8sClient client.Client, frpClient *frpcv1.Client) error {
	configMap, err := builder.NewConfigMapBuilder(k8sClient, frpClient).SetName(frpClient.Name).SetNamespace(frpClient.Namespace).Build(ctx)
	if err != nil {
		return err
	}
	if err := k8sClient.Get(ctx, client.ObjectKey{Name: frpClient.Name, Namespace: frpClient.Namespace}, &corev1.ConfigMap{}); err != nil {
		if apierrors.IsNotFound(err) {
			return k8sClient.Create(ctx, configMap)
		}
		return err
	} else {
		if err := k8sClient.Update(ctx, configMap); err != nil {
			return err
		}
	}
	return nil
}

func tryCreateRbac(ctx context.Context, k8sClient client.Client, namespace string, serviceAccountName string, roleName string, bindingName string) error {
	br := builder.NewRbacBuilder(namespace, serviceAccountName, roleName, bindingName)
	serviceAccount := br.BuildServiceAccount()
	if err := k8sClient.Get(ctx, client.ObjectKey{Name: serviceAccountName, Namespace: namespace}, &corev1.ServiceAccount{}); apierrors.IsNotFound(err) {
		if err := k8sClient.Create(ctx, serviceAccount); err != nil {
			return err
		}
	}
	role := br.BuildRole()
	if err := k8sClient.Get(ctx, client.ObjectKey{Name: roleName, Namespace: namespace}, &rbacv1.Role{}); apierrors.IsNotFound(err) {
		if err := k8sClient.Create(ctx, role); err != nil {
			return err
		}
	}
	binding := br.BuildRoleBinding()
	if err := k8sClient.Get(ctx, client.ObjectKey{Name: bindingName, Namespace: namespace}, &rbacv1.RoleBinding{}); apierrors.IsNotFound(err) {
		if err := k8sClient.Create(ctx, binding); err != nil {
			return err
		}
	}
	return nil
}
