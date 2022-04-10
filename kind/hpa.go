package kind

import (
	"encoding/json"

	"github.com/ghodss/yaml"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	hpaApiVersion = "autoscaling/v1"
	hpaKind       = "HorizontalPodAutoscaler"
)

type hpa struct {
	Name                     string `json:"name" binding:"required"`
	MinReplicas              int    `json:"min_replicas" binding:"required"`
	MaxReplicas              int    `json:"max_replicas" binding:"required"`
	CPUUtilizationPercentage int    `json:"cpu_utilization_percentage"`
}

func NewHpa(name string, minReplicas, maxReplicas, cPUUtilizationPercentage int) *hpa {
	return &hpa{
		Name:                     name,
		MinReplicas:              minReplicas,
		MaxReplicas:              maxReplicas,
		CPUUtilizationPercentage: cPUUtilizationPercentage,
	}
}

// RenderYaml return yaml
func (h *hpa) RenderYaml() (bytes []byte, err error) {
	r := h.render()
	bytes, err = json.Marshal(r)
	if err != nil {
		return
	}
	return yaml.JSONToYAML(bytes)
}

func (h *hpa) render() *autoscalingv1.HorizontalPodAutoscaler {
	return &autoscalingv1.HorizontalPodAutoscaler{
		TypeMeta: metav1.TypeMeta{
			APIVersion: hpaApiVersion,
			Kind:       hpaKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: h.Name,
		},
		Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{
				APIVersion: rolloutApiVersion,
				Kind:       rolloutKind,
				Name:       h.Name,
			},
			MinReplicas:                    int32Ptr(int32(h.MinReplicas)),
			MaxReplicas:                    int32(h.MaxReplicas),
			TargetCPUUtilizationPercentage: int32Ptr(int32(h.CPUUtilizationPercentage)),
		},
	}
}

// DeserializeHpa 反序列化 Hpa
func DeserializeHpa(b []byte) (mp map[string]interface{}, err error) {
	bytes, err := yaml.YAMLToJSON(b)
	if err != nil {
		return
	}
	as := &autoscalingv1.HorizontalPodAutoscaler{}
	err = json.Unmarshal(bytes, &as)
	if err != nil {
		return
	}

	mp = map[string]interface{}{
		"hpa_min_replicas":               as.Spec.MinReplicas,
		"hpa_max_replicas":               as.Spec.MaxReplicas,
		"hpa_cpu_utilization_percentage": as.Spec.TargetCPUUtilizationPercentage,
		"is_hpa_enable":                  true,
	}
	return
}
