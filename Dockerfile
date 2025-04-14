# 第一阶段：构建应用程序
FROM golang:1.24 AS builder

WORKDIR /app

# 复制 Go 模块定义
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用程序，静态链接
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o k8s-terminal-go .

# 第二阶段：创建最终镜像
FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# 从构建阶段复制二进制文件
COPY --from=builder /app/k8s-terminal-go .

# 设置环境变量
ENV PORT=8080

# 暴露端口
EXPOSE 8080

# 运行应用程序
CMD ["./k8s-terminal-go"] 