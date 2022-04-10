package kind

import (
	"encoding/json"
	"time"

	"github.com/ghodss/yaml"
	"github.com/zhengyansheng/common"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	deploymentApiVersion = "apps/v1"
	deploymentKind       = "Deployment"
)

type deployment struct {
	Name                         string          `json:"name" binding:"required"`
	ProjectEnv                   string          `json:"project_env" binding:"required"`
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

type ContainerPort struct {
	Name string `json:"name"`
	Port int    `json:"port"`
}

type ContainerEnv struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Label struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func NewDeployment() *deployment {
	return &deployment{}
}

// RenderYaml return yaml
func (deploy *deployment) RenderYaml() (bytes []byte, err error) {
	r := deploy.render()
	bytes, err = json.Marshal(r)
	if err != nil {
		return
	}
	return yaml.JSONToYAML(bytes)
}

// render return deployment struct
func (deploy *deployment) render() *appsv1.Deployment {
	// Set
	var (
		containerPorts []apiv1.ContainerPort
		containerEnvs  []apiv1.EnvVar
		nodeSelector   = make(map[string]string)
	)
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

	containerEnvs = []apiv1.EnvVar{
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
		{Name: "spring.profiles.active", Value: deploy.ProjectEnv},
		{Name: "env", Value: deploy.ProjectEnv},
	}
	for _, env := range deploy.ContainerEnvs {
		containerEnvs = append(containerEnvs, apiv1.EnvVar{Name: env.Key, Value: env.Value})
	}

	maxSurge := intstr.Parse(deploy.MaxSurge)
	maxUnavailable := intstr.Parse(deploy.MaxUnavailable)

	// generator
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: deploymentApiVersion,
			Kind:       deploymentKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: deploy.Name,
			Labels: map[string]string{
				"app": deploy.Name,
			},
		},
		Spec: appsv1.DeploymentSpec{
			RevisionHistoryLimit: int32Ptr(int32(revisionHistoryLimit)),
			//Replicas: int32Ptr(int32(deploy.Replicas)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": deploy.Name,
				},
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxSurge:       &maxSurge,
					MaxUnavailable: &maxUnavailable,
				},
			},
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
							Name:    deploy.Name,
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
								FailureThreshold:    int32(deploy.ReadinessFailureThreshold),
								InitialDelaySeconds: int32(deploy.ReadinessInitialDelaySeconds),
								PeriodSeconds:       int32(deploy.ReadinessPeriodSeconds),
								SuccessThreshold:    int32(deploy.ReadinessSuccessThreshold),
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
								FailureThreshold:    int32(deploy.LivenessFailureThreshold),
								InitialDelaySeconds: int32(deploy.LivenessInitialDelaySeconds),
								PeriodSeconds:       int32(deploy.LivenessPeriodSeconds),
								SuccessThreshold:    int32(deploy.LivenessSuccessThreshold),
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
								{Name: "logs", MountPath: "/opt/app/logs", SubPath: "$(POD_NAME)"},
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
						{Name: "logs", VolumeSource: apiv1.VolumeSource{
							HostPath: &apiv1.HostPathVolumeSource{
								Path: "/tmp/",
							},
						}},
					},
				},
			},
		},
	}
}

// DeserializeDeployment 反序列化 deployment
func DeserializeDeployment(b []byte) (mp map[string]interface{}, err error) {
	var (
		containerEnvs      []map[string]interface{}
		retainNodeSelector []map[string]interface{}
	)

	bytes, err := yaml.YAMLToJSON(b)
	if err != nil {
		return
	}
	dpl := &appsv1.Deployment{}
	err = json.Unmarshal(bytes, &dpl)
	if err != nil {
		return
	}

	for _, envVar := range dpl.Spec.Template.Spec.Containers[0].Env {
		if ok := common.Contains(defaultEnvKeys, envVar.Name); !ok {
			m := make(map[string]interface{})
			m["key"] = envVar.Name
			m["value"] = envVar.Value
			containerEnvs = append(containerEnvs, m)
		}
	}

	for k, v := range dpl.Spec.Template.Spec.NodeSelector {
		m := make(map[string]interface{})
		m["key"] = k
		m["value"] = v
		retainNodeSelector = append(retainNodeSelector, m)
	}

	// container
	container := dpl.Spec.Template.Spec.Containers[0]

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
		"name":                            dpl.Name,
		"start_cmd":                       container.Command[2],
		"pre_stop":                        container.Lifecycle.PreStop.Exec.Command[2],
		"post_start":                      container.Lifecycle.PostStart.Exec.Command[2],
		"min_cpu":                         minCpuFloat,
		"min_mem":                         minMemFloat,
		"max_cpu":                         maxCpuFloat,
		"max_mem":                         maxMemFloat,
		"health_check_path":               container.ReadinessProbe.Handler.HTTPGet.Path,
		"health_check_port":               container.ReadinessProbe.Handler.HTTPGet.Port.IntVal,
		"max_surge":                       dpl.Spec.Strategy.RollingUpdate.MaxSurge.String(),
		"max_unavailable":                 dpl.Spec.Strategy.RollingUpdate.MaxUnavailable.String(),
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
		"termination_grace":               *dpl.Spec.Template.Spec.TerminationGracePeriodSeconds,
		"container_envs":                  containerEnvs,
		"node_selector":                   retainNodeSelector,
	}
	return
}
