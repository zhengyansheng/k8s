package kind

import (
	"encoding/json"
	"fmt"

	"github.com/ghodss/yaml"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	configMapApiVersion = "v1"
	configMapKind       = "ConfigMap"
)

type configMap struct {
	Name     string            `json:"name" binding:"required"`
	FileName string            `json:"file_name"`
	Data     map[string]string `json:"data"`
}

func NewConfigMap(name, fileName string, data map[string]string) *configMap {
	return &configMap{
		Name:     name,
		FileName: fileName,
		Data:     data,
	}
}

// RenderYaml return yaml
func (cm *configMap) RenderYaml() (bytes []byte, err error) {
	r := cm.render()
	bytes, err = json.Marshal(r)
	if err != nil {
		return
	}
	return yaml.JSONToYAML(bytes)
}

// render return configmap struct
func (cm *configMap) render() *apiv1.ConfigMap {
	content := ""
	for k, v := range cm.Data {
		content += fmt.Sprintf("%v: %v\n", k, v)
	}
	return &apiv1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: configMapApiVersion,
			Kind:       configMapKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: cm.Name,
			Labels: map[string]string{
				"app": cm.Name,
			},
		},
		Data: map[string]string{
			cm.FileName: content,
		},
	}
}

// DeserializationConfigMap 反序列化 configmap
func DeserializationConfigMap(b []byte) (mp map[string]string, err error) {
	bytes, err := yaml.YAMLToJSON(b)
	if err != nil {
		return
	}
	cfgMap := &apiv1.ConfigMap{}
	err = json.Unmarshal(bytes, &cfgMap)
	if err != nil {
		return
	}
	return cfgMap.Data, nil
}
