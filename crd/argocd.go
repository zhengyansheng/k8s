package crd

import (
	"context"
	"fmt"

	argocdclient "github.com/argoproj/argo-cd/v2/pkg/apiclient"
	applicationpkg "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	argoappv1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	argoio "github.com/argoproj/argo-cd/v2/util/io"
)

type argoCDServer struct {
	ServerAddr string `json:"server_addr"`
	ServerPort int    `json:"server_port"`
	AuthToken  string `json:"auth_token"`
	AppName    string `json:"app_name"`
	Revision   string `json:"revision"`
}

func NewArgoServer(argoHost, authToken, appName string) *argoCDServer {
	return &argoCDServer{
		ServerAddr: argoHost,
		ServerPort: 80,
		AuthToken:  authToken,
		AppName:    appName,
		Revision:   "HEAD",
	}
}

// Sync 手动同步application
// Sync https://github.com/argoproj/argo-cd/blob/691b77ff9eb3c6e274debd5f1cba32414416b56b/cmd/argocd/commands/app.go#L1311
func (s *argoCDServer) Sync() (*argoappv1.Application, error) {
	clientOpts := argocdclient.ClientOptions{
		ServerAddr: fmt.Sprintf("%s:%d", s.ServerAddr, s.ServerPort),
		Insecure:   true,
		//PortForwardNamespace: "argocd",
		//PortForward: true,
		AuthToken: s.AuthToken,
	}

	acdClient := argocdclient.NewClientOrDie(&clientOpts)
	conn, appIf := acdClient.NewApplicationClientOrDie()
	defer argoio.Close(conn)

	syncOptionsFactory := func() *applicationpkg.SyncOptions {
		syncOptions := applicationpkg.SyncOptions{}
		items := make([]string, 0)
		if len(items) == 0 {
			// for prevent send even empty array if not need
			return nil
		}
		syncOptions.Items = items
		return &syncOptions
	}

	syncReq := applicationpkg.ApplicationSyncRequest{
		Name:        &s.AppName,
		DryRun:      false,
		Revision:    s.Revision,
		Resources:   nil,
		Prune:       false,
		SyncOptions: syncOptionsFactory(),
	}
	syncReq.Strategy = &argoappv1.SyncStrategy{Apply: &argoappv1.SyncStrategyApply{}}
	syncReq.Strategy.Apply.Force = false
	ctx := context.Background()
	return appIf.Sync(ctx, &syncReq)
}
