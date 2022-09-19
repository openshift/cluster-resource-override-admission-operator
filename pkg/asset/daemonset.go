package asset

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func (a *Asset) DaemonSet() *daemonset {
	return &daemonset{
		asset: a,
	}
}

type daemonset struct {
	asset *Asset
}

func (d *daemonset) Name() string {
	return d.asset.Values().Name
}

func (d *daemonset) New() *appsv1.DaemonSet {
	tolerationSeconds := int64(120)
	values := d.asset.Values()

	return &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: values.Namespace,
			Name:      d.Name(),
			Labels: map[string]string{
				values.OwnerLabelKey: values.OwnerLabelValue,
			},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					values.SelectorLabelKey: values.SelectorLabelValue,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: d.Name(),
					Labels: map[string]string{
						values.SelectorLabelKey: values.SelectorLabelValue,
						values.OwnerLabelKey:    values.OwnerLabelValue,
					},
				},
				Spec: corev1.PodSpec{
					HostNetwork: true,
					NodeSelector: map[string]string{
						"node-role.kubernetes.io/master": "",
					},
					ServiceAccountName: values.ServiceAccountName,
					Containers: []corev1.Container{
						{
							Name:            d.Name(),
							Image:           values.OperandImage,
							ImagePullPolicy: corev1.PullAlways,
							Command: []string{
								"/usr/bin/cluster-resource-override-admission",
							},
							Args: []string{
								"--secure-port=9400",
								"--bind-address=127.0.0.1",
								"--tls-cert-file=/var/serving-cert/tls.crt",
								"--tls-private-key-file=/var/serving-cert/tls.key",
								"--v=3",
							},
							Env: []corev1.EnvVar{
								{
									Name:  "CONFIGURATION_PATH",
									Value: "/etc/clusterresourceoverride/config/override.yaml",
								},
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 9400,
									HostPort:      9400,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: pointer.BoolPtr(false),
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{"ALL"},
								},
								RunAsNonRoot: pointer.BoolPtr(true),
								SeccompProfile: &corev1.SeccompProfile{
									Type: "RuntimeDefault",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "serving-cert",
									MountPath: "/var/serving-cert",
								},
								{
									Name:      "configuration",
									MountPath: "/etc/clusterresourceoverride/config/override.yaml",
									SubPath:   values.ConfigurationKey,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "serving-cert",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: d.asset.ServiceServingSecret().Name(),
									DefaultMode: func() *int32 {
										v := int32(420)
										return &v
									}(),
								},
							},
						},

						{
							Name: "configuration",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: d.asset.Configuration().Name(),
									},
								},
							},
						},
					},
					Tolerations: []corev1.Toleration{
						{
							Key:      "node-role.kubernetes.io/master",
							Operator: corev1.TolerationOpExists,
							Effect:   corev1.TaintEffectNoSchedule,
						},
						{
							Key:               "node.kubernetes.io/unreachable",
							Operator:          corev1.TolerationOpExists,
							Effect:            corev1.TaintEffectNoExecute,
							TolerationSeconds: &tolerationSeconds,
						},
						{
							Key:               "node.kubernetes.io/not-ready",
							Operator:          corev1.TolerationOpExists,
							Effect:            corev1.TaintEffectNoExecute,
							TolerationSeconds: &tolerationSeconds,
						},
					},
				},
			},
		},
	}
}
