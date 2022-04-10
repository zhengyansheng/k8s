package api

import (
	"context"
	"encoding/json"
	"errors"

	argocdapp "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

type dynamicClient struct {
	DynamicClient   dynamic.Interface
	DiscoveryClient *discovery.DiscoveryClient
	KubeConfig      *rest.Config
}

// NewDynamicClient 初始化 dynamic client
func NewDynamicClient(kubeConfig string) (dc *dynamicClient, err error) {
	c, err := GetK8sDiscoveryClient(kubeConfig)
	if err != nil {
		return
	}
	return &dynamicClient{
		DynamicClient: c,
	}, nil
}

// Create create a kind resource
func (c *dynamicClient) Create(b []byte) (*unstructured.Unstructured, error) {
	u, mp, err := c.render(b)
	if err != nil {
		return nil, err
	}
	resREST, err := c.resourceREST(u, mp)
	if err != nil {
		return nil, err
	}

	return resREST.Create(context.TODO(), u, metav1.CreateOptions{})
}

// Update update a kind resource
func (c *dynamicClient) Update(b []byte) (*unstructured.Unstructured, error) {
	u, mp, err := c.render(b)
	if err != nil {
		return nil, err
	}
	resREST, err := c.resourceREST(u, mp)
	if err != nil {
		return nil, err
	}

	return resREST.Update(context.TODO(), u, metav1.UpdateOptions{})

}

// DynamicGet get crd resource
func (c *dynamicClient) DynamicGet(apiVersion, kind, namespace, name string) (*unstructured.Unstructured, error) {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": apiVersion,
			"kind":       kind,
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
		},
	}
	bytes, err := json.Marshal(obj)
	u, mp, err := c.render(bytes)
	if err != nil {
		return nil, err
	}
	resREST, err := c.resourceREST(u, mp)
	if err != nil {
		return nil, err
	}
	return resREST.Get(context.TODO(), name, metav1.GetOptions{})
}

// GetCrdRolloutV1alpha1 get crd rollout struct
func (c *dynamicClient) GetCrdRolloutV1alpha1(namespace, name string) (r v1alpha1.Rollout, err error) {
	uns, err := c.DynamicGet("argoproj.io/v1alpha1", "Rollout", namespace, name)
	if err != nil {
		return
	}
	b, err := json.Marshal(uns)
	if err != nil {
		return
	}
	if err = json.Unmarshal(b, &r); err != nil {
		return
	}
	return
}

// Get get a kind resource
func (c *dynamicClient) Get(b []byte, name string) (*unstructured.Unstructured, error) {
	u, mp, err := c.render(b)
	if err != nil {
		return nil, err
	}
	resREST, err := c.resourceREST(u, mp)
	if err != nil {
		return nil, err
	}
	return resREST.Get(context.TODO(), name, metav1.GetOptions{})
}

// Delete delete a kind resource
func (c *dynamicClient) Delete(b []byte, name string) error {
	u, mp, err := c.render(b)
	if err != nil {
		return err
	}
	resREST, err := c.resourceREST(u, mp)
	if err != nil {
		return err
	}

	return resREST.Delete(context.TODO(), name, metav1.DeleteOptions{})

}

// DynamicPatch apply resource
func (c *dynamicClient) DynamicPatch(namespace string, name string) (*unstructured.Unstructured, error) {
	opts := metav1.PatchOptions{}
	unpausePatch := `{
	"spec": {
		"paused": false
	},
	"status": {
		"pauseConditions": null
	}
}`
	//data := []byte(fmt.Sprintf(`{"spec":{"paused": true}}`))
	data := []byte(unpausePatch)
	deploymentRes := schema.GroupVersionResource{Group: "argoproj.io", Version: "v1alpha1", Resource: "rollouts"}
	return c.DynamicClient.Resource(deploymentRes).Namespace(namespace).Patch(context.TODO(), name, types.MergePatchType, data, opts)
}

func (c *dynamicClient) render(b []byte) (*unstructured.Unstructured, *meta.RESTMapping, error) {
	/*
		仅支持一个文件一个资源
	*/

	// 获取支持的资源类型列表
	resources, err := restmapper.GetAPIGroupResources(c.DiscoveryClient)
	if err != nil {
		return nil, nil, err
	}

	// 创建 'Discovery REST Mapper'，获取查询的资源的类型
	mapper := restmapper.NewDiscoveryRESTMapper(resources)

	runtimeObject, groupVersionAndKind, err := yaml.NewDecodingSerializer(
		unstructured.UnstructuredJSONScheme).Decode(b, nil, nil)

	if err != nil {
		return nil, nil, err
	}

	// 查找 Group/Version/Kind 的 REST 映射
	mapping, err := mapper.RESTMapping(groupVersionAndKind.GroupKind(), groupVersionAndKind.Version)
	if err != nil {
		return nil, nil, err
	}

	// 转换 yaml 的类型为 Unstructured
	unstructuredObj, ok := runtimeObject.(*unstructured.Unstructured)
	if !ok {
		err = errors.New("yaml serializer can't type assertion (*unstructured.Unstructured)")
		return nil, nil, err
	}

	return unstructuredObj, mapping, nil
}

func (c *dynamicClient) resourceREST(u *unstructured.Unstructured, mp *meta.RESTMapping) (dynamic.ResourceInterface, error) {
	// 需要为 namespace 范围内的资源提供不同的接口
	if mp.Scope.Name() == meta.RESTScopeNameNamespace {
		if u.GetNamespace() == "" {
			u.SetNamespace("default")
		}
		return c.DynamicClient.Resource(mp.Resource).Namespace(u.GetNamespace()), nil
	} else {
		return c.DynamicClient.Resource(mp.Resource), nil
	}
}

// GetArgoApplicationV1alpha1 get crd application struct
func (c *dynamicClient) GetArgoApplicationV1alpha1(name string) (r argocdapp.Application, err error) {
	uns, err := c.DynamicGet("argoproj.io/v1alpha1", "Application", "argocd", name)
	if err != nil {
		return
	}
	b, err := json.Marshal(uns)
	if err != nil {
		return
	}
	if err = json.Unmarshal(b, &r); err != nil {
		return
	}
	return
}
