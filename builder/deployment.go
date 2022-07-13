package builder

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DeployBuilder struct {
	Name      string
	Namespace string
	Image     string
}

func NewDeployBuilder() *DeployBuilder {
	return &DeployBuilder{}
}

func (n *DeployBuilder) SetName(name string) *DeployBuilder {
	n.Name = name
	return n
}

func (n *DeployBuilder) SetNamespace(namespace string) *DeployBuilder {
	n.Namespace = namespace
	return n
}

func (n *DeployBuilder) SetImage(image string) *DeployBuilder {
	n.Image = image
	return n
}

func (n *DeployBuilder) Build() *appsv1.Deployment {
	runAsUser := int64(1000)
	runAsGroup := int64(1000)
	readOnlyRootFilesystem := true
	allowPrivilegeEscalation := false
	replicas := int32(1)
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      n.Name,
			Namespace: n.Namespace,
			Labels:    n.BuildLabels(),
			Annotations: map[string]string{
				"sidecar.istio.io/inject":                "false",
				"linkerd.io/inject":                      "disabled",
				"kuma.io/sidecar-injection":              "disabled",
				"appmesh.k8s.aws/sidecarInjectorWebhook": "disabled",
				"injector.nsm.nginx.com/auto-inject":     "false",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: n.BuildLabels(),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: n.BuildLabels(),
					Annotations: map[string]string{
						"sidecar.istio.io/inject":                "false",
						"linkerd.io/inject":                      "disabled",
						"kuma.io/sidecar-injection":              "disabled",
						"appmesh.k8s.aws/sidecarInjectorWebhook": "disabled",
						"injector.nsm.nginx.com/auto-inject":     "false",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "frpc",
					// TODO need InitContainers ?
					// InitContainers: []corev1.Container{
					// 	{
					// 		Name:    "init-config",
					// 		Image:   "busybox:latest",
					// 		Command: []string{"touch", "/frp/config.ini"},
					// 		VolumeMounts: []corev1.VolumeMount{
					// 			{
					// 				Name:      "config",
					// 				MountPath: "/frp",
					// 			},
					// 		},
					// 	},
					// },
					Containers: []corev1.Container{
						{
							Name:  "config-reload",
							Image: "kiwigrid/k8s-sidecar:1.15.0",
							SecurityContext: &corev1.SecurityContext{
								RunAsUser:                &runAsUser,
								RunAsGroup:               &runAsGroup,
								ReadOnlyRootFilesystem:   &readOnlyRootFilesystem,
								AllowPrivilegeEscalation: &allowPrivilegeEscalation,
							},
							Env: []corev1.EnvVar{
								{
									Name: "POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"},
									},
								},
								{
									Name:  "LABEL",
									Value: n.Name + "-config-as-code",
								},
								{
									Name:  "FOLDER",
									Value: "/frp",
								},
								{
									Name:  "NAMESPACE",
									Value: n.Namespace,
								},
								{
									Name:  "REQ_URL",
									Value: "http://localhost:7400/api/reload", // TODO
								},
								{
									Name:  "REQ_METHOD",
									Value: "GET",
								},
								{
									Name:  "REQ_USERNAME",
									Value: "frpc-admin",
								},
								{
									Name:  "REQ_PASSWORD",
									Value: "frpc-password",
								},
								{
									Name:  "REQ_RETRY_CONNECT",
									Value: "10", // TODO
								},
								{
									Name:  "SKIP_TLS_VERIFY",
									Value: "true", // TODO
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/frp",
								},
							},
						},
						{
							Name:    "frpc",
							Image:   n.Image,
							Command: []string{"frpc", "-c", "/frp/config.ini"},
							Ports: []corev1.ContainerPort{
								{ContainerPort: int32(4040)},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/frp",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}

	return deploy
}

func (n *DeployBuilder) BuildLabels() map[string]string {
	var labels = map[string]string{
		"app.kubernetes.io/name":       n.Name,
		"app.kubernetes.io/managed-by": "frpc-operator",
		"app.kubernetes.io/created-by": n.Name,
	}

	return labels
}
