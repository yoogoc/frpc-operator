package builder

import (
	"context"

	frpcv1 "github.com/YoogoC/frpc-operator/api/v1"
	"github.com/YoogoC/frpc-operator/gen"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ConfigMapBuilder struct {
	Name      string
	Namespace string
	k8sClient client.Client
	frpClient *frpcv1.Client
}

func NewConfigMapBuilder(k8sClient client.Client, frpClient *frpcv1.Client) *ConfigMapBuilder {
	return &ConfigMapBuilder{
		k8sClient: k8sClient,
		frpClient: frpClient,
	}
}

func (builder *ConfigMapBuilder) SetName(name string) *ConfigMapBuilder {
	builder.Name = name
	return builder
}

func (builder *ConfigMapBuilder) SetNamespace(namespace string) *ConfigMapBuilder {
	builder.Namespace = namespace
	return builder
}

func (builder *ConfigMapBuilder) Build(ctx context.Context) (*corev1.ConfigMap, error) {
	var proxyList frpcv1.ProxyList
	if err := builder.k8sClient.List(ctx, &proxyList, client.InNamespace(builder.Namespace)); err != nil {
		return nil, err
	}
	var proxies []frpcv1.Proxy
	for _, item := range proxyList.Items {
		if item.Spec.Client == builder.Name && item.DeletionTimestamp == nil {
			proxies = append(proxies, item)
		}
	}

	configData, err := gen.Gen(builder.k8sClient, builder.frpClient, proxies)
	if err != nil {
		return nil, err
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      builder.Name,
			Namespace: builder.Namespace,
			Labels: map[string]string{
				"app":                            builder.Name,
				"generated":                      "frpc-operator",
				builder.Name + "-config-as-code": "yes",
			},
		},
		Data: map[string]string{"config.ini": configData},
	}, nil
}
