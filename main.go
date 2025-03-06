package main

import (
	"fmt"
	"k8s-terminal-go/config"
	"k8s-terminal-go/handlers"
	"k8s-terminal-go/k8s"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化 K8s 客户端
	k8sClient, err := k8s.NewClient(cfg.KubeConfigPath)
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	// 创建 Gin 路由
	r := gin.Default()

	// 配置 CORS
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// 终端 WebSocket 处理
	terminalHandler := handlers.NewTerminalHandler(k8sClient)
	r.GET("/api/terminal", terminalHandler.HandleTerminal)

	// 启动服务器
	log.Printf("Starting server on :%d", cfg.Port)
	if err := r.Run(fmt.Sprintf(":%d", cfg.Port)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
