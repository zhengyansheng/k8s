package kind

import (
	"encoding/json"

	argocdapp "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/health"
	"github.com/ghodss/yaml"
	"github.com/zhengyansheng/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	argoApplicationApiVersion = "argoproj.io/v1alpha1"
	argoApplicationKind       = "Application"
	argoLocalCluster          = "https://kubernetes.default.svc"
)

type application struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	RepoURL   string `json:"repo_url"`
	RepoPath  string `json:"url_path"`
}

func NewApplication(name, repoUrl, repoPath string) *application {
	return &application{
		Name:      name,
		Namespace: "argocd",
		RepoURL:   repoUrl,
		RepoPath:  repoPath,
	}
}

// RenderYaml return yaml
func (a *application) RenderYaml() (bytes []byte, err error) {
	r := a.render()
	bytes, err = json.Marshal(r)
	if err != nil {
		return
	}
	return yaml.JSONToYAML(bytes)
}

// render return deployment struct
func (a *application) render() *argocdapp.Application {
	return &argocdapp.Application{
		TypeMeta: metav1.TypeMeta{
			Kind:       argoApplicationKind,
			APIVersion: argoApplicationApiVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      a.Name,
			Namespace: a.Namespace,
		},
		Spec: argocdapp.ApplicationSpec{
			Source: argocdapp.ApplicationSource{
				RepoURL:        a.RepoURL,
				Path:           a.RepoPath,
				TargetRevision: "HEAD",
			},
			Destination: argocdapp.ApplicationDestination{
				Server:    argoLocalCluster,
				Namespace: a.Namespace,
			},
			// https://argo-cd.readthedocs.io/en/stable/user-guide/auto_sync/
			SyncPolicy: &argocdapp.SyncPolicy{
				Automated: &argocdapp.SyncPolicyAutomated{
					Prune:      true,
					SelfHeal:   true,
					AllowEmpty: true,
				},
				SyncOptions: []string{"CreateNamespace=true"},
			},
			Project: "default",
		},
	}
}

type ApplicationSummary struct {
	Name          string                  `json:"name"`
	Images        []string                `json:"images"`
	HealthStatus  health.HealthStatusCode `json:"health_status"`
	StartedAt     string                  `json:"started_at"`
	HealthMessage string                  `json:"health_message"`
	IsPause       bool                    `json:"is_pause"`
	IsReady       bool                    `json:"is_ready"`
	IsFailed      bool                    `json:"is_failed"`
}

// DeserializationApplication 反序列化 application
func DeserializationApplication(data []byte) (as ApplicationSummary, err error) {
	var (
		isFailed  bool
		isReady   bool
		isPause   bool
		startedAt = "0000-00-00 00:00:00"
	)
	applicationConfig := &argocdapp.Application{}
	err = json.Unmarshal(data, &applicationConfig)
	if err != nil {
		return
	}

	switch applicationConfig.Status.Health.Status {
	case health.HealthStatusMissing:
		startedAt = applicationConfig.Status.OperationState.StartedAt.Format(common.TimeFormat) // 上次变更时间
	case health.HealthStatusHealthy:
		isReady = true
	case health.HealthStatusDegraded:
		isFailed = true
	case health.HealthStatusSuspended:
		isPause = true
	}

	return ApplicationSummary{
		Name:          applicationConfig.Name,
		Images:        applicationConfig.Status.Summary.Images,
		HealthStatus:  applicationConfig.Status.Health.Status,
		StartedAt:     startedAt, // 上次变更时间
		HealthMessage: applicationConfig.Status.Health.Message,
		IsPause:       isPause,  // 暂停
		IsReady:       isReady,  // 准备就绪
		IsFailed:      isFailed, // 出现 Degraded 状态
	}, nil
}
