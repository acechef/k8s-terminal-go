package config

import (
	"os"
	"path/filepath"
	"strconv"
)

// Config 存储应用配置
type Config struct {
	KubeConfigPath string
	Port           int
}

// LoadConfig 从环境变量加载配置
func LoadConfig() (*Config, error) {
	// 默认使用 $HOME/.kube/config
	kubeConfigPath := os.Getenv("KUBECONFIG")
	if kubeConfigPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		kubeConfigPath = filepath.Join(homeDir, ".kube", "config")
	}

	// 默认端口 8080
	port := 8080
	if portStr := os.Getenv("PORT"); portStr != "" {
		var err error
		port, err = strconv.Atoi(portStr)
		if err != nil {
			return nil, err
		}
	}

	return &Config{
		KubeConfigPath: kubeConfigPath,
		Port:           port,
	}, nil
}
