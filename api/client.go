package api

import (
	"errors"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// GetK8sClient 获取k8s clientSet Client
func GetK8sClient(k8sConf string) (*kubernetes.Clientset, error) {
	cfg, err := initClient(k8sConf)
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(cfg)
}

// GetK8sDiscoveryClient 获取k8s discovery Client
func GetK8sDiscoveryClient(k8sConf string) (dynamic.Interface, error) {
	cfg, err := initClient(k8sConf)
	if err != nil {
		return nil, err
	}

	return dynamic.NewForConfig(cfg)
}

func initClient(k8sConf string) (*rest.Config, error) {
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(k8sConf))
	// skips the validity check for the server's certificate. This will make your HTTPS connections insecure.
	// config.TLSClientConfig.Insecure = true
	if err != nil {
		return nil, errors.New("KubeConfig内容错误")
	}
	return config, nil
}
