package api

const (
	CrashLoopBackOff                    PodStatus = "CrashLoopBackOff"
	InvalidImageName                    PodStatus = "InvalidImageName"
	ImageInspectError                   PodStatus = "ImageInspectError"
	ErrImageNeverPull                   PodStatus = "ErrImageNeverPull"
	ImagePullBackOff                    PodStatus = "ImagePullBackOff"
	RegistryUnavailable                 PodStatus = "RegistryUnavailable"
	ErrImagePull                        PodStatus = "ErrImagePull"
	CreateContainerConfigError          PodStatus = "CreateContainerConfigError"
	CreateContainerError                PodStatus = "CreateContainerError"
	mInternalLifecyclePreStartContainer PodStatus = "m.internalLifecycle.PreStartContainer"
	RunContainerError                   PodStatus = "RunContainerError"
	PostStartHookError                  PodStatus = "PostStartHookError"
	ContainersNotInitialized            PodStatus = "ContainersNotInitialized"
	ContainersNotReady                  PodStatus = "ContainersNotReady"
	ContainerCreating                   PodStatus = "ContainerCreating"
	PodInitializing                     PodStatus = "PodInitializing"
	DockerDaemonNotReady                PodStatus = "DockerDaemonNotReady"
	NetworkPluginNotReady               PodStatus = "NetworkPluginNotReady"
	Terminating                         PodStatus = "Terminating"
)
