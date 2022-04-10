package api

type PodStatus string

type PodEvent struct {
	PodName string `json:"pod_name"`
	Message string `json:"message"`
	Note    string `json:"note"`
}

type PodInfo struct {
	Status       PodStatus   `json:"status"`
	Age          string      `json:"age"`
	Ready        string      `json:"ready"`
	PodIP        string      `json:"pod_ip"`
	PodName      string      `json:"pod_name"`
	RestartCount int32       `json:"restart_count"`
	HostIP       string      `json:"host_ip"`
	Tag          string      `json:"tag"`
	StartTime    string      `json:"start_time"`
	Message      string      `json:"message"`
	Events       []*PodEvent `json:"events"`
	Containers   []Container `json:"containers"`
}

type Container struct {
	ContainerID  string `json:"containerID"`
	Name         string `json:"name"`
	Ready        bool   `json:"ready"`
	RestartCount int32  `json:"restartCount"`
	StartTime    string `json:"startTime"`
}

type DeploymentInfo struct {
	Name                string `json:"name"`
	Age                 string `json:"age"`
	AvailableReplicas   int32  `json:"available_replicas"`
	ReadyReplicas       int32  `json:"ready_replicas"`
	UnavailableReplicas int32  `json:"unavailable_replicas"`
	Replicas            int32  `json:"replicas"`
	Phase               string `json:"phase"`
}

type Namespace struct {
	Name  string
	Phase string
	Age   string
}

type Node struct {
	Name             string            `json:"name"`
	Status           interface{}       `json:"status"`
	Roles            string            `json:"roles"`
	Cpu              string            `json:"cpu"`
	Mem              string            `json:"mem"`
	Age              string            `json:"age"`
	Version          string            `json:"version"`
	InternalIp       string            `json:"internal_ip"`
	ExternalIp       string            `json:"external_ip"`
	OsImage          string            `json:"os_image"`
	KernelVersion    string            `json:"kernel_version"`
	ContainerRuntime string            `json:"container_runtime"`
	Labels           map[string]string `json:"labels"`
}

type Events struct {
	Namespace string `json:"namespace"`
	LastSeen  string `json:"last_seen"`
	Type      string `json:"type"`
	REASON    string `json:"reason"`
	Object    string `json:"object"`
	Message   string `json:"message"`
}
