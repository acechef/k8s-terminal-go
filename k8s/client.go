package k8s

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client 封装 Kubernetes 客户端
type Client struct {
	ClientSet *kubernetes.Clientset
	Config    *rest.Config
}

// NewClient 创建新的 Kubernetes 客户端
func NewClient(kubeConfigPath string) (*Client, error) {
	// 尝试从提供的 kubeconfig 路径加载配置
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		// 如果失败，尝试使用集群内配置（用于在 Pod 中运行的情况）
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	}

	// 创建 clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &Client{
		ClientSet: clientset,
		Config:    config,
	}, nil
}
