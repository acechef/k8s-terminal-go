# Kubernetes Terminal Go

这是一个基于Go语言的Kubernetes终端Web应用，允许用户通过Web界面访问Kubernetes集群中的Pod终端。

## 功能特性

- 通过WebSocket连接到Kubernetes Pod终端
- 支持多种认证方式连接到Kubernetes集群
- 可配置的端口和Kubeconfig路径

## 依赖条件

- Go 1.16+
- Kubernetes集群访问权限

## 配置

应用程序通过环境变量进行配置：

- `KUBECONFIG`：Kubernetes配置文件路径（默认为`$HOME/.kube/config`）
- `PORT`：应用程序监听端口（默认为`8080`）

## Docker构建和运行

### 构建Docker镜像

```bash
docker build -t k8s-terminal-go:latest .
```

### 运行Docker镜像

#### 使用本地kubeconfig运行（开发环境）

```bash
docker run -p 8080:8080 -v $HOME/.kube:/root/.kube k8s-terminal-go:latest
```

#### 在Kubernetes集群内部运行（使用服务账号）

无需挂载kubeconfig，应用程序将使用服务账号凭证：

```bash
docker run -p 8080:8080 k8s-terminal-go:latest
```

### 自定义端口

```bash
docker run -p 9000:9000 -e PORT=9000 -v $HOME/.kube:/root/.kube k8s-terminal-go:latest
```

## API端点

- `GET /api/terminal` - WebSocket终端连接

## 部署到Kubernetes

1. 构建并推送镜像到镜像仓库
2. 应用Kubernetes部署配置

```bash
# 构建镜像
docker build -t your-registry.com/k8s-terminal-go:latest .

# 推送镜像
docker push your-registry.com/k8s-terminal-go:latest

# 部署到集群
kubectl apply -f deployment.yaml
```

## 许可证

[MIT](LICENSE) 