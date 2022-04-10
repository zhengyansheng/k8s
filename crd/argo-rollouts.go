package crd

import (
	"fmt"
	"os"

	roclientset "github.com/argoproj/argo-rollouts/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-rollouts/pkg/kubectl-argo-rollouts/cmd/abort"
	"github.com/argoproj/argo-rollouts/pkg/kubectl-argo-rollouts/cmd/promote"
	"github.com/argoproj/argo-rollouts/pkg/kubectl-argo-rollouts/options"
	"github.com/zhengyansheng/k8s/api"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

/*
	# References
	https://github.com/argoproj/argo-rollouts/blob/master/cmd/kubectl-argo-rollouts/main.go
	https://github.com/argoproj/argo-rollouts/blob/master/pkg/kubectl-argo-rollouts/cmd/cmd.go#L65
	https://github.com/argoproj/argo-rollouts/blob/master/pkg/kubectl-argo-rollouts/cmd/promote/promote.go#L68
*/

type ArgoClient struct {
	rolloutsClient *roclientset.Clientset
}

func NewArgoClient(kubeConfig string) (*ArgoClient, error) {
	dc, err := api.NewDynamicClient(kubeConfig)
	if err != nil {
		return nil, err
	}
	rolloutsClient, err := roclientset.NewForConfig(dc.KubeConfig)
	if err != nil {
		return &ArgoClient{}, err
	}
	return &ArgoClient{rolloutsClient: rolloutsClient}, nil
}

// Promote 推动继续
func (ac *ArgoClient) Promote(namespace, name string) error {
	rolloutIf := ac.rolloutsClient.ArgoprojV1alpha1().Rollouts(namespace)
	ro, err := promote.PromoteRollout(rolloutIf, name, false, false, false)
	if err != nil {
		return err
	}
	fmt.Printf("ro: %+v\n", ro)
	return nil
}

// Abort 中止
func (ac *ArgoClient) Abort(namespace, name string) error {
	streams := genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}
	o := options.NewArgoRolloutsOptions(streams)
	rolloutIf := o.RolloutsClientset().ArgoprojV1alpha1().Rollouts(namespace)
	ro, err := abort.AbortRollout(rolloutIf, name)
	if err != nil {
		return err
	}
	fmt.Printf("ro: %+v\n", ro)
	return nil
}
