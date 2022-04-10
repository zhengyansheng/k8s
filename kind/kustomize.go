package kind

import (
	"encoding/json"
	"fmt"

	"github.com/ghodss/yaml"
	"sigs.k8s.io/kustomize/api/types"
)

type kustomizationConfig struct {
	Name              string            `json:"name"`              // 项目名称
	Namespace         string            `json:"namespace"`         // 名称空间
	Count             int               `json:"count"`             // 副本数
	Resources         []string          `json:"resources"`         // 引用基础资源目录
	NamePrefix        string            `json:"namePrefix"`        // 资源统一前缀
	PatchMergeFiles   []string          `json:"patch_merge_files"` // 合并的文件
	CommonAnnotations map[string]string `json:"commonAnnotations"` // 注解
	CommonLabels      map[string]string `json:"commonLabels"`      // 标签
	Version           string            `json:"version"`           // 版本
	ImageRegistry     string            `json:"image_registry"`    // 镜像仓库
	Configurations    []string          `json:"configurations"`    // 转换器配置 // https://argoproj.github.io/argo-rollouts/features/kustomize/
	OpenAPI           map[string]string `json:"open_api"`          // https://github.com/argoproj/argo-rollouts/blob/master/docs/features/kustomize/rollout_cr_schema.json
}

func NewKustomizationConfig(name, namespace string, count int, imageRegistry, version string) *kustomizationConfig {
	return &kustomizationConfig{
		Name:          name,
		Namespace:     namespace,
		Count:         count,
		ImageRegistry: imageRegistry,
		Version:       version,
	}
}

// RenderYaml return yaml
func (k *kustomizationConfig) RenderYaml() (bytes []byte, err error) {
	r := k.render()
	bytes, err = json.Marshal(r)
	if err != nil {
		return
	}
	return yaml.JSONToYAML(bytes)
}

func (k *kustomizationConfig) render() *types.Kustomization {
	var patchesStrategicMerge []types.PatchStrategicMerge
	for _, file := range k.PatchMergeFiles {
		patchesStrategicMerge = append(patchesStrategicMerge, types.PatchStrategicMerge(file))
	}

	// return
	return &types.Kustomization{
		TypeMeta: types.TypeMeta{
			Kind:       ConstKustomizeVersion,
			APIVersion: ConstKustomizeApiVersion,
		},
		MetaData:              &types.ObjectMeta{},
		Resources:             k.Resources,
		NamePrefix:            k.NamePrefix,
		Namespace:             k.Namespace,
		CommonLabels:          k.CommonLabels,
		CommonAnnotations:     k.CommonAnnotations,
		PatchesStrategicMerge: patchesStrategicMerge,
		Images: []types.Image{
			{
				Name:    k.Name,
				NewName: fmt.Sprintf("%s/%s", k.ImageRegistry, k.Name),
				NewTag:  k.Version,
			},
		},
		Replicas: []types.Replica{
			{
				Name:  k.Name,
				Count: toInt64(k.Count),
			},
		},
		Configurations: k.Configurations,
		OpenAPI:        k.OpenAPI,
	}
}

// DeserializeKustomizationConfig 反序列化 KustomizationConfig
func DeserializeKustomizationConfig(b []byte) (mp map[string]interface{}, err error) {
	var (
		newTag      string
		isHpaEnable bool
		replicas    int64
	)
	bytes, err := yaml.YAMLToJSON(b)
	if err != nil {
		return
	}
	kus := &types.Kustomization{}
	err = json.Unmarshal(bytes, &kus)
	if err != nil {
		return
	}

	if len(kus.Images) == 0 {
		newTag = "latest"
	} else {
		newTag = kus.Images[0].NewTag
	}
	for _, resource := range kus.Resources {
		if resource == "hpa.yaml" {
			isHpaEnable = true
		}
	}
	if len(kus.Replicas) != 0 {
		replicas = kus.Replicas[0].Count
	}
	mp = map[string]interface{}{
		"replicas":      replicas,
		"tag":           newTag,
		"node_labels":   kus.CommonLabels,
		"is_hpa_enable": isHpaEnable,
	}
	return
}
