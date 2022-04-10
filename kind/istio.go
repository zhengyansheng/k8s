package kind

import (
	"encoding/json"

	"github.com/ghodss/yaml"
	"istio.io/api/networking/v1beta1"
	istiov1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	istioApiVersion = "networking.istio.io/v1beta1"
	istioKind       = "VirtualService"
)

type virtualService struct {
	Name     string   `json:"name"`
	Hosts    []string `json:"hosts"`
	Gateways []string `json:"gateways"`
	JiraID   string   `json:"jira_id"`
}

func NewVirtualService(name string, hosts, gateways []string) *virtualService {
	return &virtualService{
		Name:     name,
		Hosts:    hosts,
		Gateways: gateways,
	}
}

// RenderYaml return yaml
func (v *virtualService) RenderYaml() (bytes []byte, err error) {
	render := v.render()
	bytes, err = json.Marshal(render)
	if err != nil {
		return
	}
	return yaml.JSONToYAML(bytes)
}

// Render return deployment struct
func (v *virtualService) render() *istiov1beta1.VirtualService {
	matchHeader := map[string]*v1beta1.StringMatch{
		"XXX-podenv": &v1beta1.StringMatch{
			MatchType:            &v1beta1.StringMatch_Exact{Exact: v.JiraID},
			XXX_NoUnkeyedLiteral: struct{}{},
			XXX_unrecognized:     nil,
			XXX_sizecache:        0,
		},
	}

	return &istiov1beta1.VirtualService{
		TypeMeta: metav1.TypeMeta{
			Kind:       istioKind,
			APIVersion: istioApiVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: v.Name,
		},
		Spec: v1beta1.VirtualService{
			Hosts:    v.Hosts,
			Gateways: v.Gateways,
			Http: []*v1beta1.HTTPRoute{
				{
					Match: []*v1beta1.HTTPMatchRequest{
						{
							Headers: matchHeader,
						},
					},
					Route: []*v1beta1.HTTPRouteDestination{
						{
							Destination: &v1beta1.Destination{
								Host: v.Name,
							},
						},
					},
				},
			},
		},
	}
}

// DeserializeVirtualService 反序列化 virtualService
func DeserializeVirtualService(b []byte) (mp map[string]interface{}, err error) {
	bytes, err := yaml.YAMLToJSON(b)
	if err != nil {
		return
	}
	vs := &istiov1beta1.VirtualService{}
	err = json.Unmarshal(bytes, &vs)
	if err != nil {
		return
	}

	mp = map[string]interface{}{
		"name":     vs.Name,
		"hosts":    vs.Spec.Hosts,
		"gateways": vs.Spec.Gateways,
	}
	return
}
