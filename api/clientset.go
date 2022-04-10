package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zhengyansheng/common"
	appsv1 "k8s.io/api/apps/v1"
	autoscallingv1 "k8s.io/api/autoscaling/v1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/api/events/v1beta1"
	networkv1 "k8s.io/api/networking/v1"
	networkbeta1 "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/yaml"
)

type clientSetClient struct {
	ClientSet  *kubernetes.Clientset
	KubeConfig *rest.Config
}

func NewClientSet(kubeConfig string) (cs *clientSetClient, err error) {
	c, err := GetK8sClient(kubeConfig)
	if err != nil {
		return
	}
	return &clientSetClient{
		ClientSet: c,
	}, nil
}

func (c *clientSetClient) defaultContext() context.Context {
	return context.TODO()
}

// DeploymentCreate 创建 deployment
func (c *clientSetClient) DeploymentCreate(namespace string, deployment *appsv1.Deployment) (*appsv1.Deployment, error) {
	return c.ClientSet.AppsV1().Deployments(namespace).Create(c.defaultContext(), deployment, metav1.CreateOptions{})
}

// DeploymentList 获取 deployment
func (c *clientSetClient) DeploymentList(namespace string) (*appsv1.DeploymentList, error) {
	return c.ClientSet.AppsV1().Deployments(namespace).List(c.defaultContext(), metav1.ListOptions{})
}

// DeploymentListFormat 获取 deployment 格式化后的数据
func (c *clientSetClient) DeploymentListFormat(namespace string) (deploymentResources []DeploymentInfo, err error) {
	dm, err := c.DeploymentList(namespace)
	if err != nil {
		return
	}
	for _, deploy := range dm.Items {
		dmInfo, err := c.DeploymentGetFormat(namespace, deploy.Name)
		if err != nil {
			return deploymentResources, err
		}
		deploymentResources = append(deploymentResources, dmInfo)
	}
	return
}

// DeploymentGet 查询单个 deployment 原生数据
func (c *clientSetClient) DeploymentGet(namespace string, name string) (*appsv1.Deployment, error) {
	return c.ClientSet.AppsV1().Deployments(namespace).Get(c.defaultContext(), name, metav1.GetOptions{})
}

// DeploymentGetFormat 查询单个 deployment 格式化数据
func (c *clientSetClient) DeploymentGetFormat(namespace string, name string) (detail DeploymentInfo, err error) {
	deploy, err := c.DeploymentGet(namespace, name)
	if err != nil {
		return
	}
	createTime := deploy.ObjectMeta.CreationTimestamp.Format(common.SecLocalTimeFormat)
	currentTime := time.Now().Format(common.SecLocalTimeFormat)
	subTime := common.SubTime(createTime, currentTime)
	detail = DeploymentInfo{
		Name:                deploy.Name,
		Age:                 subTime,
		UnavailableReplicas: deploy.Status.UnavailableReplicas, // 不可用副本数
		Replicas:            deploy.Status.Replicas,            // 期望副本数
		ReadyReplicas:       deploy.Status.ReadyReplicas,       // 正常副本数
		AvailableReplicas:   deploy.Status.AvailableReplicas,   // 可用副本数
		Phase:               "Healthy",
	}
	return
}

func (c *clientSetClient) StatefulSetList(namespace string) (*appsv1.StatefulSetList, error) {
	return c.ClientSet.AppsV1().StatefulSets(namespace).List(c.defaultContext(), metav1.ListOptions{})
}

func (c *clientSetClient) DaemonSetList(namespace string) (*appsv1.DaemonSetList, error) {
	return c.ClientSet.AppsV1().DaemonSets(namespace).List(c.defaultContext(), metav1.ListOptions{})
}

// PodEventsGet 解析 pod/event 的详细信息
func (c *clientSetClient) PodEventsGet(namespace string, name string) (podEvents []PodInfo, err error) {
	labelSelector := map[string]string{"app": name}
	podList, err := c.Pods(namespace, labelSelector)
	if err != nil {
		return
	}
	for _, pod := range podList.Items {
		podInfo, err := c.PodDetail(namespace, pod.Name)
		if err != nil {
			return podEvents, err
		}

		eventList, err := c.EventsList(namespace)
		if err != nil {
			return podEvents, err
		}
		for _, item := range eventList.Items {
			if item.Regarding.Name == pod.Name {
				e := &PodEvent{
					PodName: pod.Name,
					Message: item.Reason,
					Note:    item.Note,
				}
				podInfo.Events = append(podInfo.Events, e)
			}
		}
		podEvents = append(podEvents, podInfo)
	}
	return
}

func (c *clientSetClient) DeploymentUpdate(namespace string, deployment *appsv1.Deployment) (*appsv1.Deployment, error) {
	return c.ClientSet.AppsV1().Deployments(namespace).Update(c.defaultContext(), deployment, metav1.UpdateOptions{})
}

func (c *clientSetClient) DeploymentDelete(ns string, name string) error {
	deletePolicy := metav1.DeletePropagationForeground
	return c.DeploymentDeleteWithOption(ns, name, c.defaultContext(), metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	})
}

func (c *clientSetClient) DeploymentDeleteContext(ctx context.Context, ns string, name string) error {
	deletePolicy := metav1.DeletePropagationForeground
	return c.DeploymentDeleteWithOption(ns, name, ctx, metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	})
}

func (c *clientSetClient) DeploymentDeleteWithOption(ns string, name string, ctx context.Context, opts metav1.DeleteOptions) error {
	deploymentClient := c.ClientSet.AppsV1().Deployments(ns)
	return deploymentClient.Delete(ctx, name, opts)
}

func (c *clientSetClient) DeploymentPods(ns string, name string) (*apiv1.PodList, error) {
	deployment, err := c.DeploymentGet(ns, name)
	if err != nil {
		return nil, err
	}
	opts := metav1.ListOptions{}
	opts.LabelSelector = labels.FormatLabels(deployment.Spec.Selector.MatchLabels)
	return c.ClientSet.CoreV1().Pods(ns).List(c.defaultContext(), opts)
}

func (c *clientSetClient) PodList(ns string) (*apiv1.PodList, error) {
	opts := metav1.ListOptions{}
	return c.ClientSet.CoreV1().Pods(ns).List(c.defaultContext(), opts)
}

func (c *clientSetClient) Pods(ns string, labelSelector map[string]string) (*apiv1.PodList, error) {
	opts := metav1.ListOptions{
		LabelSelector: labels.FormatLabels(labelSelector),
	}
	return c.ClientSet.CoreV1().Pods(ns).List(c.defaultContext(), opts)
}

// PodDelete 删除单个Pod
func (c *clientSetClient) PodDelete(ns, podName string) error {
	opts := metav1.DeleteOptions{}
	return c.ClientSet.CoreV1().Pods(ns).Delete(c.defaultContext(), podName, opts)
}

// PodGet 查询Pod信息
func (c *clientSetClient) PodGet(namespace, podName string) (*apiv1.Pod, error) {
	opts := metav1.GetOptions{}
	return c.ClientSet.CoreV1().Pods(namespace).Get(c.defaultContext(), podName, opts)
}

// PodDetail 查询Pod 详细信息
func (c *clientSetClient) PodDetail(namespace, podName string) (p PodInfo, err error) {
	pod, err := c.PodGet(namespace, podName)
	if err != nil {
		return
	}
	p = PodInfo{
		PodName: pod.Name,
		PodIP:   pod.Status.PodIP,
		HostIP:  pod.Status.HostIP,
		Tag:     strings.Split(pod.Spec.Containers[0].Image, ":")[1],
	}

	if pod.ObjectMeta.DeletionTimestamp != nil {
		p.Status = Terminating
	} else {
		p.Status = PodStatus(pod.Status.Phase)
		if pod.Status.Phase == apiv1.PodRunning {
			p.StartTime = pod.Status.StartTime.Format(common.SecLocalTimeFormat)
		}
	}

	containerCount := len(pod.Status.ContainerStatuses)
	var containerReadyCount int
	for _, containerStatuses := range pod.Status.ContainerStatuses {
		if containerStatuses.Ready == true {
			containerReadyCount++
		}
		// container
		c := Container{
			ContainerID:  containerStatuses.ContainerID,
			Name:         containerStatuses.Name,
			Ready:        containerStatuses.Ready,
			RestartCount: containerStatuses.RestartCount,
		}
		if containerStatuses.Ready {
			c.StartTime = containerStatuses.State.Running.StartedAt.Format(common.SecLocalTimeFormat)
		}
		p.Containers = append(p.Containers, c)
	}
	// sort
	optContainerSlice := p.Containers
	sort.Slice(optContainerSlice, func(i, j int) bool {
		return optContainerSlice[i].StartTime < optContainerSlice[j].StartTime
	})
	p.Containers = optContainerSlice

	p.Ready = fmt.Sprintf("%d/%d", containerReadyCount, containerCount)

	for _, cs := range pod.Status.ContainerStatuses {
		p.RestartCount = cs.RestartCount
		switch {
		case cs.State.Waiting != nil:
			p.Status = PodStatus(cs.State.Waiting.Reason)
		}
	}

	var deltaVal int64
	if pod.Status.StartTime == nil {
		deltaVal = time.Now().Unix()
	} else {
		deltaVal = pod.Status.StartTime.Unix()
	}
	p.Age = common.RuntimeAge(time.Now().Unix() - deltaVal)
	return
}

func (c *clientSetClient) PodYaml(ns string, podName string) (*apiv1.Pod, error) {
	opts := metav1.GetOptions{}
	return c.ClientSet.CoreV1().Pods(ns).Get(c.defaultContext(), podName, opts)
}

// PodExec inPod exec command
func (c *clientSetClient) PodExec(namespace, podName, container, command string) (output string, err error) {
	url := c.ClientSet.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&apiv1.PodExecOptions{
			Container: container,
			Command:   []string{"sh", "-c", command},
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec).URL()

	var stdout, stderr bytes.Buffer
	exec, err := remotecommand.NewSPDYExecutor(c.KubeConfig, "POST", url)
	if err != nil {
		return
	}
	if err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: &stdout,
		Stderr: &stderr,
	}); err != nil {
		return
	}
	if len(strings.TrimSpace(stderr.String())) == 0 {
		return strings.TrimSpace(stdout.String()), nil
	}
	return "", errors.New(strings.TrimSpace(stderr.String()))

}

// PodTTY inPod exec command
func (c *clientSetClient) PodTTY(namespace, podName, container, shellType string, conn *websocket.Conn, cols, rows uint16) (err error) {
	url := c.ClientSet.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&apiv1.PodExecOptions{
			Container: container,
			Command:   []string{shellType},
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, scheme.ParameterCodec).URL()

	exec, err := remotecommand.NewSPDYExecutor(c.KubeConfig, "POST", url)
	if err != nil {
		return
	}
	term := NewWebTerminal(conn, cols, rows)
	if err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  term,
		Stdout: term,
		Stderr: term,
		Tty:    true,
	}); err != nil {
		return
	}
	return
}

// PodLogs 查看Pod日志
func (c *clientSetClient) PodLogs(namespace string, podName, containerName string, follow bool) (io.ReadCloser, error) {
	tailLines := int64(50)
	opts := &apiv1.PodLogOptions{
		Follow:    follow, // 对应kubectl logs -f参数
		TailLines: &tailLines,
	}
	if containerName != "" {
		opts.Container = containerName
	}
	return c.ClientSet.CoreV1().Pods(namespace).GetLogs(podName, opts).Stream(c.defaultContext())
}

func (c *clientSetClient) NamespaceList() (*apiv1.NamespaceList, error) {
	return c.NamespaceListWithOption(c.defaultContext(), metav1.ListOptions{})
}

func (c *clientSetClient) NamespaceWithContentList(ctx context.Context) (*apiv1.NamespaceList, error) {
	return c.NamespaceListWithOption(ctx, metav1.ListOptions{})
}

func (c *clientSetClient) NamespaceListWithOption(ctx context.Context, opts metav1.ListOptions) (*apiv1.NamespaceList, error) {
	return c.ClientSet.CoreV1().Namespaces().List(ctx, opts)
}

func (c *clientSetClient) NamespaceGet(name string) (*apiv1.Namespace, error) {
	return c.ClientSet.CoreV1().Namespaces().Get(c.defaultContext(), name, metav1.GetOptions{})
}

func (c *clientSetClient) ServiceCreate(namespace string, service *apiv1.Service) (*apiv1.Service, error) {
	opts := metav1.CreateOptions{}
	return c.ClientSet.CoreV1().Services(namespace).Create(c.defaultContext(), service, opts)
}

func (c *clientSetClient) ServiceGet(namespace, name string) (*apiv1.Service, error) {
	opts := metav1.GetOptions{}
	return c.ClientSet.CoreV1().Services(namespace).Get(c.defaultContext(), name, opts)
}

func (c *clientSetClient) ServiceCreateYaml(namespace, serviceYaml string) (*apiv1.Service, error) {
	var svc apiv1.Service
	svcBytes, err := yaml.YAMLToJSON([]byte(serviceYaml))
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(svcBytes, &svc)
	if err != nil {
		return nil, err
	}
	return c.ServiceCreate(namespace, &svc)
}

func (c *clientSetClient) ServiceUpdate(namespace string, service *apiv1.Service) (*apiv1.Service, error) {
	opts := metav1.UpdateOptions{}
	return c.ClientSet.CoreV1().Services(namespace).Update(c.defaultContext(), service, opts)
}

func (c *clientSetClient) ServiceUpdateYaml(namespace, serviceYaml string) (*apiv1.Service, error) {
	var svc apiv1.Service
	svcBytes, err := yaml.YAMLToJSON([]byte(serviceYaml))
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(svcBytes, &svc)
	if err != nil {
		return nil, err
	}
	opts := metav1.UpdateOptions{}
	return c.ClientSet.CoreV1().Services(namespace).Update(c.defaultContext(), &svc, opts)
}

func (c *clientSetClient) ServiceDelete(namespace, name string) error {
	deletePolicy := metav1.DeletePropagationForeground
	opts := metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}
	return c.ClientSet.CoreV1().Services(namespace).Delete(c.defaultContext(), name, opts)
}

func (c *clientSetClient) IngressCreate(namespace string, ingress *networkv1.Ingress) (*networkv1.Ingress, error) {
	opts := metav1.CreateOptions{}
	return c.ClientSet.NetworkingV1().Ingresses(namespace).Create(c.defaultContext(), ingress, opts)
}

func (c *clientSetClient) IngressGetByBeta1(namespace, name string) (*networkbeta1.Ingress, error) {
	opts := metav1.GetOptions{}
	return c.ClientSet.NetworkingV1beta1().Ingresses(namespace).Get(c.defaultContext(), name, opts)
}

func (c *clientSetClient) IngressGet(namespace, name string) (*networkv1.Ingress, error) {
	opts := metav1.GetOptions{}
	return c.ClientSet.NetworkingV1().Ingresses(namespace).Get(c.defaultContext(), name, opts)
}

func (c *clientSetClient) IngressList(namespace string) (*networkv1.IngressList, error) {
	opts := metav1.ListOptions{}
	return c.ClientSet.NetworkingV1().Ingresses(namespace).List(c.defaultContext(), opts)
}

func (c *clientSetClient) IngressCreateYaml(namespace, ingressYaml string) (*networkv1.Ingress, error) {
	var networkIngress networkv1.Ingress
	svcBytes, err := yaml.YAMLToJSON([]byte(ingressYaml))
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(svcBytes, &networkIngress)
	if err != nil {
		return nil, err
	}
	return c.IngressCreate(namespace, &networkIngress)
}

func (c *clientSetClient) IngressUpdate(namespace string, ingress *networkv1.Ingress) (*networkv1.Ingress, error) {
	opts := metav1.UpdateOptions{}
	return c.ClientSet.NetworkingV1().Ingresses(namespace).Update(c.defaultContext(), ingress, opts)
}

func (c *clientSetClient) IngressDelete(namespace, name string) error {
	deletePolicy := metav1.DeletePropagationForeground
	opts := metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}
	return c.ClientSet.ExtensionsV1beta1().Ingresses(namespace).Delete(c.defaultContext(), name, opts)
}

func (c *clientSetClient) SecretCreate(namespace string, secret *apiv1.Secret) (*apiv1.Secret, error) {
	opts := metav1.CreateOptions{}
	return c.ClientSet.CoreV1().Secrets(namespace).Create(c.defaultContext(), secret, opts)
}

func (c *clientSetClient) SecretDelete(namespace, name string) error {
	opts := metav1.DeleteOptions{}
	return c.ClientSet.CoreV1().Secrets(namespace).Delete(c.defaultContext(), name, opts)
}

func (c *clientSetClient) PvcList(namespace string) (*apiv1.PersistentVolumeClaimList, error) {
	opts := metav1.ListOptions{}
	return c.ClientSet.CoreV1().PersistentVolumeClaims(namespace).List(c.defaultContext(), opts)
}

func (c *clientSetClient) ConfigmapGet(namespace, name string) (*apiv1.ConfigMap, error) {
	opts := metav1.GetOptions{}
	return c.ClientSet.CoreV1().ConfigMaps(namespace).Get(c.defaultContext(), name, opts)
}

func (c *clientSetClient) ConfigmapCreate(namespace string, cm *apiv1.ConfigMap) (*apiv1.ConfigMap, error) {
	opts := metav1.CreateOptions{}
	return c.ClientSet.CoreV1().ConfigMaps(namespace).Create(c.defaultContext(), cm, opts)
}

func (c *clientSetClient) HpaGet(namespace, name string) (*autoscallingv1.HorizontalPodAutoscaler, error) {
	opts := metav1.GetOptions{}
	return c.ClientSet.AutoscalingV1().HorizontalPodAutoscalers(namespace).Get(c.defaultContext(), name, opts)

}

func (c *clientSetClient) EventsList(namespace string) (*v1beta1.EventList, error) {
	// kubectl get events -A --sort-by=.metadata.creationTimestamp
	// v1beta1.EventList
	opts := metav1.ListOptions{}
	return c.ClientSet.EventsV1beta1().Events(namespace).List(c.defaultContext(), opts)
}

// EventsDetail 格式化后的Event清单
func (c *clientSetClient) EventsDetail() (eventList []Events, err error) {
	// kubectl get events -A --sort-by=.metadata.creationTimestamp
	// v1beta1.EventList
	ns, err := c.NamespaceList()
	if err != nil {
		return
	}
	for _, item := range ns.Items {
		eventsList, err := c.ClientSet.EventsV1beta1().Events(item.Namespace).List(c.defaultContext(), metav1.ListOptions{})
		if err != nil {
			return eventList, err
		}
		for _, el := range eventsList.Items {
			eventList = append(eventList, Events{
				Namespace: el.Namespace,
				LastSeen:  el.CreationTimestamp.Format(common.SecLocalTimeFormat),
				REASON:    el.Reason,
				Type:      el.Type,
				Message:   el.Note,
				Object:    fmt.Sprintf("%s/%s", el.Regarding.Kind, el.Regarding.Name),
			})
		}
	}
	return
}

// NodeGet 获取指定Node
func (c *clientSetClient) NodeGet(nodeName string) (*apiv1.Node, error) {
	opts := metav1.GetOptions{}
	return c.ClientSet.CoreV1().Nodes().Get(c.defaultContext(), nodeName, opts)
}

// NodeListFormat 获取所有Node
func (c *clientSetClient) NodeListFormat() (nodes []Node, err error) {
	nodeList, err := c.ClientSet.CoreV1().Nodes().List(c.defaultContext(), metav1.ListOptions{})
	if err != nil {
		return
	}

	for _, item := range nodeList.Items {
		createTime := item.CreationTimestamp.Format(common.SecLocalTimeFormat)
		nodes = append(nodes, Node{
			Name:             item.Name,
			Age:              common.SubTime(createTime, common.Now()),
			KernelVersion:    item.Status.NodeInfo.KernelVersion,
			OsImage:          item.Status.NodeInfo.OSImage,
			Version:          item.Status.NodeInfo.KubeletVersion,
			Cpu:              item.Status.Capacity.Cpu().String(),
			Mem:              item.Status.Allocatable.Memory().String(),
			Status:           item.Status.Conditions[len(item.Status.Conditions)-1].Type,
			ContainerRuntime: item.Status.NodeInfo.ContainerRuntimeVersion,
			Labels:           item.Labels,
		})
	}
	return
}

// DeploymentPause 暂停Deployment升级
func (c *clientSetClient) DeploymentPause(namespace, name string) (*appsv1.Deployment, error) {
	data := []byte(fmt.Sprintf(`{"spec":{"paused": true}}`))
	return c.deploymentPatch(namespace, c.defaultContext(), name, types.MergePatchType, data, metav1.PatchOptions{})
}

// DeploymentResume 恢复Deployment升级
func (c *clientSetClient) DeploymentResume(namespace, name string) (*appsv1.Deployment, error) {
	data := []byte(fmt.Sprintf(`{"spec":{"paused": false}}`))
	return c.deploymentPatch(namespace, c.defaultContext(), name, types.MergePatchType, data, metav1.PatchOptions{})
}

// DeploymentUpdateReplicas 修改 deployment 副本数
func (c *clientSetClient) DeploymentUpdateReplicas(namespace, name string, replicas int) (*appsv1.Deployment, error) {
	data := map[string]map[string]int{
		"spec": {
			"replicas": replicas,
		},
	}
	byteMarshal, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return c.deploymentPatch(namespace, c.defaultContext(), name, types.MergePatchType, byteMarshal, metav1.PatchOptions{})
}

func (c *clientSetClient) deploymentPatch(namespace string, ctx context.Context, name string,
	pt types.PatchType, data []byte, opts metav1.PatchOptions, subResources ...string) (*appsv1.Deployment, error) {
	return c.ClientSet.AppsV1().Deployments(namespace).Patch(ctx, name, pt, data, opts, subResources...)
}
