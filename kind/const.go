package kind

const (
	ConstServiceApiVersion = "v1"
	ConstServiceVersion    = "v1"

	//ConstIngressApiVersion = "extensions/v1beta1"
	ConstIngressApiVersion = "networking.k8s.io/v1"

	ConstKustomizeApiVersion = "kustomize.config.k8s.io/v1beta1"
	ConstKustomizeVersion    = "Kustomization"
)

var (
	defaultEnvKeys = []string{
		"APP_NAME", "TIME", "POD_NAME", "POD_IP", "NODE_NAME", "NODE_IP",
	}
)

const (
	revisionHistoryLimit = 3
)
