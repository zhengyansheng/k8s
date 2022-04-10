package kind

import (
	"encoding/json"

	"github.com/ghodss/yaml"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type service struct {
	Name        string `json:"name" binding:"required"`
	NetworkType string `json:"network_type" binding:"required"`
	PortName    string `json:"port_name" binding:"required"` // svc:name, ing:servicePort
	ExposePort  int    `json:"expose_port" binding:"required"`
	TargetPort  int    `json:"target_port" binding:"required"`
}

func NewService(name, networkType, portName string, exposePort, targetPort int) *service {
	return &service{
		Name:        name,
		NetworkType: networkType,
		PortName:    portName,
		ExposePort:  exposePort,
		TargetPort:  targetPort,
	}
}

// Render return service struct
func (svc *service) render() *apiv1.Service {
	// convert
	var serviceType apiv1.ServiceType
	switch svc.NetworkType {
	case "ClusterIP":
		serviceType = apiv1.ServiceTypeClusterIP
	case "LoadBalancer":
		serviceType = apiv1.ServiceTypeLoadBalancer
	default:
		serviceType = ""
	}

	// return
	return &apiv1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: ConstServiceApiVersion,
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: svc.Name,
			Labels: map[string]string{
				"app":        svc.Name,
				"prometheus": "true",
			},
		},
		Spec: apiv1.ServiceSpec{
			Type: serviceType,
			Selector: map[string]string{
				"app": svc.Name,
			},
			Ports: []apiv1.ServicePort{
				{
					Name:     svc.PortName,
					Port:     toInt32(svc.ExposePort),
					Protocol: apiv1.ProtocolTCP,
					TargetPort: intstr.IntOrString{
						IntVal: toInt32(svc.TargetPort),
					},
				},
			},
		},
	}
}

// RenderYaml return yaml
func (svc *service) RenderYaml() (bytes []byte, err error) {
	r := svc.render()
	bytes, err = json.Marshal(r)
	if err != nil {
		return
	}
	return yaml.JSONToYAML(bytes)
}

// DeserializeService 反序列化 service
func DeserializeService(b []byte) (mp map[string]interface{}, err error) {
	bytes, err := yaml.YAMLToJSON(b)
	if err != nil {
		return
	}
	service := &apiv1.Service{}
	err = json.Unmarshal(bytes, &service)
	if err != nil {
		return
	}

	mp = map[string]interface{}{
		"svc_network_type": service.Spec.Type,
		"svc_port_name":    service.Spec.Ports[0].Name,
		"svc_expose_port":  service.Spec.Ports[0].Port,
		"svc_target_port":  service.Spec.Ports[0].TargetPort.IntVal,
	}
	return
}
