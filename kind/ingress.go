package kind

import (
	"encoding/json"

	"github.com/ghodss/yaml"
	networkv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ingress struct {
	Name              string `json:"name" binding:"required"`
	IngressController string `json:"ingress_controller" binding:"required"`
	Domain            string `json:"domain" binding:"required"`
	Path              string `json:"path" binding:"required"`
	PortName          string `json:"port_name" binding:"required"`
	PortNumber        int    `json:"port_number" binding:"required"`
}

func NewIngress(name, ingressController, domain, path, portName string, portNumber int) *ingress {
	return &ingress{
		Name:              name,
		IngressController: ingressController,
		Domain:            domain,
		Path:              path,
		PortName:          portName,
		PortNumber:        portNumber,
	}
}

// RenderYaml return yaml
func (ingress *ingress) RenderYaml() (bytes []byte, err error) {
	r := ingress.render()
	bytes, err = json.Marshal(r)
	if err != nil {
		return
	}
	return yaml.JSONToYAML(bytes)
}

func (ingress *ingress) render() *networkv1.Ingress {
	pathType := networkv1.PathTypePrefix
	return &networkv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			APIVersion: ConstIngressApiVersion,
			Kind:       "Ingress",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: ingress.Name,
			Labels: map[string]string{
				"app": ingress.Name,
			},
			Annotations: map[string]string{
				"kubernetes.io/ingress.class": ingress.IngressController,
			},
		},
		Spec: networkv1.IngressSpec{
			Rules: []networkv1.IngressRule{
				{
					Host: ingress.Domain,
					IngressRuleValue: networkv1.IngressRuleValue{
						HTTP: &networkv1.HTTPIngressRuleValue{
							Paths: []networkv1.HTTPIngressPath{
								{
									Path:     ingress.Path,
									PathType: &pathType,
									Backend: networkv1.IngressBackend{
										Service: &networkv1.IngressServiceBackend{
											Name: ingress.Name,
											Port: networkv1.ServiceBackendPort{
												Number: toInt32(ingress.PortNumber),
											},
										},
									},
								},
							},
						},
					}},
			},
		},
	}
}

// DeserializeIngress 反序列化 ingress
func DeserializeIngress(b []byte) (mp map[string]interface{}, err error) {
	bytes, err := yaml.YAMLToJSON(b)
	if err != nil {
		return
	}
	ing := &networkv1.Ingress{}
	err = json.Unmarshal(bytes, &ing)
	if err != nil {
		return
	}

	mp = map[string]interface{}{
		"ingress_controller":  ing.ObjectMeta.Annotations["kubernetes.io/ingress.class"],
		"ingress_domain":      ing.Spec.Rules[0].Host,
		"ingress_path":        ing.Spec.Rules[0].IngressRuleValue.HTTP.Paths[0].Path,
		"ingress_port_name":   ing.Spec.Rules[0].IngressRuleValue.HTTP.Paths[0].Backend.Service.Port.Name,
		"ingress_port_number": ing.Spec.Rules[0].IngressRuleValue.HTTP.Paths[0].Backend.Service.Port.Number,
	}
	return
}
