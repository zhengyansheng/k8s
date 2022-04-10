package kind

import (
	"encoding/json"

	"github.com/ghodss/yaml"
	"istio.io/api/networking/v1beta1"
	istiov1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IGateway istio gateway
type IGateway struct {
	Name     string            `json:"name"`     // feature-gateway
	Port     uint32            `json:"port"`     // 80
	Protocol string            `json:"protocol"` // HTTP
	Hosts    []string          `json:"hosts"`    // []string{"*}
	Selector map[string]string `json:"selector"` // map[string]string{"istio": "ingressgateway"}
}

// Render return deployment struct
func (v *IGateway) Render() *istiov1beta1.Gateway {
	return &istiov1beta1.Gateway{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "networking.istio.io/v1beta1",
			Kind:       "Gateway",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: v.Name,
		},
		Spec: v1beta1.Gateway{
			Selector: v.Selector,
			Servers: []*v1beta1.Server{
				{
					Hosts: v.Hosts,
					Port: &v1beta1.Port{
						Name:     v.Name,
						Number:   v.Port,
						Protocol: v.Protocol,
					},
				},
			},
		},
	}
}

// RenderYaml return yaml
func (v *IGateway) RenderYaml() (bytes []byte, err error) {
	render := v.Render()
	bytes, err = json.Marshal(render)
	if err != nil {
		return
	}
	return yaml.JSONToYAML(bytes)
}
