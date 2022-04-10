package kind

import (
	"encoding/json"

	"github.com/ghodss/yaml"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	secretApiVersion = "v1"
	secretKind       = "Secret"
)

type secret struct {
	Name       string            `json:"name"`
	Namespace  string            `json:"namespace"`
	Labels     map[string]string `json:"labels"`
	StringData map[string]string `json:"string_data"`
}

func NewSecret(name, namespace string, labels, stringData map[string]string) *secret {
	return &secret{
		Name:       name,
		Namespace:  namespace,
		Labels:     labels,
		StringData: stringData,
	}
}

// RenderYaml return yaml
func (r *secret) RenderYaml() (bytes []byte, err error) {
	render := r.render()
	bytes, err = json.Marshal(render)
	if err != nil {
		return
	}
	return yaml.JSONToYAML(bytes)
}

// Render return deployment struct
func (r *secret) render() *apiv1.Secret {
	return &apiv1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: secretApiVersion,
			Kind:       secretKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.Name,
			Namespace: r.Namespace,
			Labels:    r.Labels,
		},
		StringData: r.StringData,
		Type:       apiv1.SecretTypeOpaque,
	}
}
