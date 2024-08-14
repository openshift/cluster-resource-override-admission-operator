package asset

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)

var tolerationSeconds = int64(120)

var DefaultReplicas int32 = 2

var DefaultNodeSelector = map[string]string{
	"node-role.kubernetes.io/master": "",
}

var DefaultTolerations = []corev1.Toleration{
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
}

func (a *Asset) Deployment() *deployment {
	return &deployment{
		asset: a,
	}
}

type deployment struct {
	asset *Asset
}

func (d *deployment) Name() string {
	return d.asset.Values().Name
}

func (d *deployment) New() *appsv1.Deployment {
	maxUnavailable := intstr.FromInt32(1)
	maxSurge := intstr.FromInt32(0)
	values := d.asset.Values()

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: values.Namespace,
			Name:      d.Name(),
			Labels: map[string]string{
				values.OwnerLabelKey: values.OwnerLabelValue,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &DefaultReplicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					values.SelectorLabelKey: values.SelectorLabelValue,
				},
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: &maxUnavailable,
					MaxSurge:       &maxSurge,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: values.Name,
					Labels: map[string]string{
						values.SelectorLabelKey: values.SelectorLabelValue,
					},
				},
				Spec: corev1.PodSpec{
					NodeSelector:       DefaultNodeSelector,
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
								"--tls-cert-file=/var/serving-cert/tls.crt",
								"--tls-private-key-file=/var/serving-cert/tls.key",
								"--v=8",
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
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/healthz",
										Port:   intstr.FromInt(9400),
										Scheme: corev1.URISchemeHTTPS,
									},
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
					Tolerations: DefaultTolerations,
					Affinity: &corev1.Affinity{
						PodAntiAffinity: &corev1.PodAntiAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
								{
									LabelSelector: &metav1.LabelSelector{
										MatchExpressions: []metav1.LabelSelectorRequirement{
											{
												Key:      values.SelectorLabelKey,
												Operator: metav1.LabelSelectorOpIn,
												Values: []string{
													values.SelectorLabelValue,
												},
											},
										},
									},
									TopologyKey: "kubernetes.io/hostname",
								},
							},
						},
					},
				},
			},
		},
	}
}
