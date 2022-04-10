package kind

import (
	"encoding/json"
	"time"

	"github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/zhengyansheng/common"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	rolloutApiVersion = "argoproj.io/v1alpha1"
	rolloutKind       = "Rollout"
)

// Rollout all fields
type Rollout struct {
	Name                         string          `json:"name" binding:"required"`
	ProjectEnv                   string          `json:"project_env" binding:"required"`
	DeployStrategy               string          `json:"deploy_strategy"` // 部署策略 RollingUpdate/ Canary
	StartCommand                 string          `json:"start_command" binding:"required"`
	PostStart                    string          `json:"post_start" binding:"required"`
	PreStop                      string          `json:"pre_stop" binding:"required"`
	MinMem                       int             `json:"min_mem" binding:"required"`
	MaxMem                       int             `json:"max_mem" binding:"required"`
	MinCpu                       string          `json:"min_cpu" binding:"required"`
	MaxCpu                       string          `json:"max_cpu" binding:"required"`
	Replicas                     int             `json:"replicas" binding:"required"`
	MaxSurge                     string          `json:"max_surge" binding:"required"`
	MaxUnavailable               string          `json:"max_unavailable" binding:"required"`
	ImagePullSecret              string          `json:"image_pull_secret" binding:"required"`
	HealthCheckPath              string          `json:"health_check_path" binding:"required"`
	HealthCheckPort              int             `json:"health_check_port" binding:"required"`
	TerminationGrace             int             `json:"termination_grace" binding:"required"` // 杀掉pod前的宽限期
	ContainerPorts               []ContainerPort `json:"container_ports" binding:"required"`
	ReadinessInitialDelaySeconds int             `json:"readiness_initial_delay_seconds" binding:"required"` // 就绪
	ReadinessPeriodSeconds       int             `json:"readiness_period_seconds" binding:"required"`
	ReadinessTimeoutSeconds      int             `json:"readiness_timeout_seconds" binding:"required"`
	ReadinessSuccessThreshold    int             `json:"readiness_success_threshold" binding:"required"`
	ReadinessFailureThreshold    int             `json:"readiness_failure_threshold" binding:"required"`
	LivenessInitialDelaySeconds  int             `json:"liveness_initial_delay_seconds" binding:"required"` // 存活
	LivenessPeriodSeconds        int             `json:"liveness_period_seconds" binding:"required"`
	LivenessTimeoutSeconds       int             `json:"liveness_timeout_seconds" binding:"required"`
	LivenessSuccessThreshold     int             `json:"liveness_success_threshold" binding:"required"`
	LivenessFailureThreshold     int             `json:"liveness_failure_threshold" binding:"required"`
	NodeLabels                   []Label         `json:"node_labels" binding:"required"`
	ContainerEnvs                []ContainerEnv  `json:"container_envs" binding:"required"`
	NodeSelector                 []Label         `json:"node_selector" binding:"required"`
	LogVolume                    LogVolume       `json:"log_volume"`
}

type LogVolume struct {
	LogHostPath    string `json:"log_host_path"`
	ConfigFileName string `json:"config_file_name"`
}

// Render return deployment struct
func (deploy *Rollout) Render() *v1alpha1.Rollout {
	// Set
	var (
		strategy       v1alpha1.RolloutStrategy
		containerPorts []apiv1.ContainerPort
		nodeSelector   map[string]string
		setWeight      = 20
	)

	maxSurge := intstr.Parse(deploy.MaxSurge)
	maxUnavailable := intstr.Parse(deploy.MaxUnavailable)

	switch deploy.DeployStrategy {
	case "rollingUpdate":
		strategy = v1alpha1.RolloutStrategy{
			Canary: &v1alpha1.CanaryStrategy{
				MaxSurge:       &maxSurge,
				MaxUnavailable: &maxUnavailable,
			},
		}
	case "canary":
		strategy = v1alpha1.RolloutStrategy{
			Canary: &v1alpha1.CanaryStrategy{
				MaxSurge:       &maxSurge,
				MaxUnavailable: &maxUnavailable,
				Steps: []v1alpha1.CanaryStep{
					{
						SetWeight: int32Ptr(int32(setWeight)),
						Pause:     &v1alpha1.RolloutPause{},
					},
				},
			},
		}
	default:
		strategy = v1alpha1.RolloutStrategy{
			Canary: &v1alpha1.CanaryStrategy{
				MaxSurge:       &maxSurge,
				MaxUnavailable: &maxUnavailable,
			},
		}
	}

	nodeSelector = make(map[string]string)
	for _, node := range deploy.NodeSelector {
		nodeSelector[node.Key] = node.Value
	}

	for _, p := range deploy.ContainerPorts {
		containerPorts = append(containerPorts, apiv1.ContainerPort{
			Name:          p.Name,
			Protocol:      apiv1.ProtocolTCP,
			ContainerPort: int32(p.Port),
		})
	}

	containerEnvs := []apiv1.EnvVar{
		{Name: "APP_NAME", Value: deploy.Name},
		{Name: "TIME", Value: time.Now().Format(common.SecLocalTimeFormat)},
		{Name: "POD_NAME", ValueFrom: &apiv1.EnvVarSource{
			FieldRef: &apiv1.ObjectFieldSelector{
				APIVersion: "v1",
				FieldPath:  "metadata.name",
			},
		}},
		{Name: "POD_IP", ValueFrom: &apiv1.EnvVarSource{
			FieldRef: &apiv1.ObjectFieldSelector{
				APIVersion: "v1",
				FieldPath:  "status.podIP",
			},
		}},
		{Name: "NODE_NAME", ValueFrom: &apiv1.EnvVarSource{
			FieldRef: &apiv1.ObjectFieldSelector{
				APIVersion: "v1",
				FieldPath:  "spec.nodeName",
			},
		}},
		{Name: "NODE_IP", ValueFrom: &apiv1.EnvVarSource{
			FieldRef: &apiv1.ObjectFieldSelector{
				APIVersion: "v1",
				FieldPath:  "status.hostIP",
			},
		}},
	}
	for _, env := range deploy.ContainerEnvs {
		containerEnvs = append(containerEnvs, apiv1.EnvVar{Name: env.Key, Value: env.Value})
	}

	// generator
	return &v1alpha1.Rollout{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rolloutApiVersion,
			Kind:       rolloutKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: deploy.Name,
			Labels: map[string]string{
				"app": deploy.Name,
			},
		},
		Spec: v1alpha1.RolloutSpec{
			//Replicas: int32Ptr(int32(deploy.Replicas)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": deploy.Name,
				},
			},
			Strategy:             strategy,
			RevisionHistoryLimit: int32Ptr(int32(revisionHistoryLimit)),
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": deploy.Name,
					},
				},
				Spec: apiv1.PodSpec{
					//Affinity: &apiv1.Affinity{
					//	PodAffinity: &apiv1.PodAffinity{
					//		PreferredDuringSchedulingIgnoredDuringExecution: []apiv1.WeightedPodAffinityTerm{
					//			{PodAffinityTerm: apiv1.PodAffinityTerm{
					//				LabelSelector: &metav1.LabelSelector{
					//					MatchExpressions: []metav1.LabelSelectorRequirement{
					//						{
					//							Key:      "app",
					//							Operator: metav1.LabelSelectorOpIn,
					//							Values:   []string{deploy.Name},
					//						},
					//						{
					//							Key:      "version",
					//							Operator: metav1.LabelSelectorOpIn,
					//							Values:   []string{deploy.Image},
					//						},
					//					},
					//				},
					//				TopologyKey: "kubernetes.io/hostname",
					//			},
					//				Weight: 100,
					//			},
					//		},
					//	},
					//},
					NodeSelector: nodeSelector,
					Containers: []apiv1.Container{
						{
							Name: deploy.Name,
							// TODO
							Image:   deploy.Name,
							Ports:   containerPorts,
							Command: []string{"sh", "-c", deploy.StartCommand},
							Lifecycle: &apiv1.Lifecycle{
								PostStart: &apiv1.Handler{
									Exec: &apiv1.ExecAction{
										Command: []string{"sh", "-c", deploy.PostStart},
									},
								},
								PreStop: &apiv1.Handler{
									Exec: &apiv1.ExecAction{
										Command: []string{"sh", "-c", deploy.PreStop},
									},
								},
							},
							ReadinessProbe: &apiv1.Probe{
								InitialDelaySeconds: int32(deploy.ReadinessInitialDelaySeconds),
								PeriodSeconds:       int32(deploy.ReadinessPeriodSeconds),
								SuccessThreshold:    int32(deploy.ReadinessSuccessThreshold),
								FailureThreshold:    int32(deploy.ReadinessFailureThreshold),
								TimeoutSeconds:      int32(deploy.ReadinessTimeoutSeconds),
								Handler: apiv1.Handler{
									HTTPGet: &apiv1.HTTPGetAction{
										Path: deploy.HealthCheckPath,
										Port: intstr.IntOrString{
											IntVal: int32(deploy.HealthCheckPort),
										},
										Scheme: "HTTP",
									}},
							},
							LivenessProbe: &apiv1.Probe{
								InitialDelaySeconds: int32(deploy.LivenessInitialDelaySeconds),
								PeriodSeconds:       int32(deploy.LivenessPeriodSeconds),
								SuccessThreshold:    int32(deploy.LivenessSuccessThreshold),
								FailureThreshold:    int32(deploy.LivenessFailureThreshold),
								TimeoutSeconds:      int32(deploy.LivenessTimeoutSeconds),
								Handler: apiv1.Handler{
									HTTPGet: &apiv1.HTTPGetAction{
										Path: deploy.HealthCheckPath,
										Port: intstr.IntOrString{
											IntVal: int32(deploy.HealthCheckPort),
										},
										Scheme: "HTTP",
									}},
							},
							VolumeMounts: []apiv1.VolumeMount{
								{
									Name:      "logs",
									MountPath: "/opt/app/logs",
									SubPath:   "$(POD_NAME)",
								},
								{
									Name:      "config-map-volume",
									MountPath: "/app/",
									ReadOnly:  true,
								},
							},
							Resources: apiv1.ResourceRequirements{
								Limits: map[apiv1.ResourceName]resource.Quantity{
									apiv1.ResourceCPU:    resource.MustParse(deploy.MaxCpu),
									apiv1.ResourceMemory: *resource.NewQuantity(int64(deploy.MaxMem*1024*1024), resource.BinarySI),
								},
								Requests: apiv1.ResourceList{
									apiv1.ResourceCPU:    resource.MustParse(deploy.MinCpu),
									apiv1.ResourceMemory: *resource.NewQuantity(int64(deploy.MinMem*1024*1024), resource.BinarySI),
								},
							},
							Env: containerEnvs,
						},
					},
					DNSConfig: &apiv1.PodDNSConfig{
						Options: []apiv1.PodDNSConfigOption{
							{Name: "ndots", Value: stringPtr("1")},
							{Name: "single-request-reopen", Value: nil},
						},
					},
					ImagePullSecrets: []apiv1.LocalObjectReference{
						{Name: deploy.ImagePullSecret},
					},
					TerminationGracePeriodSeconds: int64Ptr(int64(deploy.TerminationGrace)),
					Volumes: []apiv1.Volume{
						{
							Name: "logs",
							VolumeSource: apiv1.VolumeSource{
								HostPath: &apiv1.HostPathVolumeSource{
									//Path: "/tmp/",
									Path: deploy.LogVolume.LogHostPath,
								},
							},
						},
						{
							Name: "config-map-volume",
							VolumeSource: apiv1.VolumeSource{
								ConfigMap: &apiv1.ConfigMapVolumeSource{
									LocalObjectReference: apiv1.LocalObjectReference{
										Name: deploy.Name,
									},
									Items: []apiv1.KeyToPath{
										{
											Key:  deploy.LogVolume.ConfigFileName,
											Path: deploy.LogVolume.ConfigFileName,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// RenderYaml return yaml
func (deploy *Rollout) RenderYaml() (bytes []byte, err error) {
	r := deploy.Render()
	bytes, err = json.Marshal(r)
	if err != nil {
		return
	}
	return yaml.JSONToYAML(bytes)
}

// Deserialize deserialize to map
func (deploy *Rollout) Deserialize(b []byte) (mp map[string]interface{}, err error) {
	var (
		containerEnvs      []map[string]interface{}
		retainNodeSelector []map[string]interface{}
	)

	bytes, err := yaml.YAMLToJSON(b)
	if err != nil {
		return
	}
	rollout := &v1alpha1.Rollout{}
	if err = json.Unmarshal(bytes, &rollout); err != nil {
		return
	}

	for _, envVar := range rollout.Spec.Template.Spec.Containers[0].Env {
		if ok := common.Contains(defaultEnvKeys, envVar.Name); !ok {
			m := make(map[string]interface{})
			m["key"] = envVar.Name
			m["value"] = envVar.Value
			containerEnvs = append(containerEnvs, m)
		}
	}

	for k, v := range rollout.Spec.Template.Spec.NodeSelector {
		m := make(map[string]interface{})
		m["key"] = k
		m["value"] = v
		retainNodeSelector = append(retainNodeSelector, m)
	}

	// container
	container := rollout.Spec.Template.Spec.Containers[0]

	// Set mem cpu
	minCpu := container.Resources.Requests.Cpu().String()
	minMem := container.Resources.Requests.Memory().String()
	MaxCpu := container.Resources.Limits.Cpu().String()
	MaxMem := container.Resources.Limits.Memory().String()
	minCpuFloat, err := common.CustomUnitGiInter(minCpu)
	if err != nil {
		return nil, err
	}
	minMemFloat, err := common.CustomUnitGiInter(minMem)
	if err != nil {
		return nil, err
	}
	maxCpuFloat, err := common.CustomUnitGiInter(MaxCpu)
	if err != nil {
		return nil, err
	}
	maxMemFloat, err := common.CustomUnitGiInter(MaxMem)
	if err != nil {
		return nil, err
	}

	// return
	mp = map[string]interface{}{
		"name":                            rollout.Name,
		"start_cmd":                       container.Command[2],
		"pre_stop":                        container.Lifecycle.PreStop.Exec.Command[2],
		"post_start":                      container.Lifecycle.PostStart.Exec.Command[2],
		"min_cpu":                         minCpuFloat,
		"min_mem":                         minMemFloat,
		"max_cpu":                         maxCpuFloat,
		"max_mem":                         maxMemFloat,
		"health_check_path":               container.ReadinessProbe.Handler.HTTPGet.Path,
		"health_check_port":               container.ReadinessProbe.Handler.HTTPGet.Port.IntVal,
		"max_surge":                       rollout.Spec.Strategy.Canary.MaxSurge,
		"max_unavailable":                 rollout.Spec.Strategy.Canary.MaxUnavailable,
		"readiness_initial_delay_seconds": container.ReadinessProbe.InitialDelaySeconds,
		"readiness_period_seconds":        container.ReadinessProbe.PeriodSeconds,
		"readiness_timeout_seconds":       container.ReadinessProbe.TimeoutSeconds,
		"readiness_success_threshold":     container.ReadinessProbe.SuccessThreshold,
		"readiness_failure_threshold":     container.ReadinessProbe.FailureThreshold,
		"liveness_initial_delay_seconds":  container.LivenessProbe.InitialDelaySeconds,
		"liveness_period_seconds":         container.LivenessProbe.PeriodSeconds,
		"liveness_timeout_seconds":        container.LivenessProbe.TimeoutSeconds,
		"liveness_success_threshold":      container.LivenessProbe.SuccessThreshold,
		"liveness_failure_threshold":      container.LivenessProbe.FailureThreshold,
		"container_port_name":             container.Ports[0].Name,
		"container_port_number":           container.Ports[0].ContainerPort,
		"termination_grace":               *rollout.Spec.Template.Spec.TerminationGracePeriodSeconds,
		"container_envs":                  containerEnvs,
		"node_selector":                   retainNodeSelector,
	}
	return
}
