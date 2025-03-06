package handlers

import (
	"encoding/json"
	"fmt"
	"k8s-terminal-go/k8s"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

// TerminalMessage 定义终端消息结构
type TerminalMessage struct {
	Op   string `json:"op"`
	Data string `json:"data"`
	Rows uint16 `json:"rows,omitempty"`
	Cols uint16 `json:"cols,omitempty"`
}

// TerminalSession 实现 PtyHandler 接口
type TerminalSession struct {
	wsConn   *websocket.Conn
	sizeChan chan remotecommand.TerminalSize
	doneChan chan struct{}
}

// NewTerminalSession 创建新的终端会话
//func NewTerminalSession(conn *websocket.Conn) *TerminalSession {
//	return &TerminalSession{
//		wsConn:   conn,
//		sizeChan: make(chan remotecommand.TerminalSize),
//		doneChan: make(chan struct{}),
//	}
//}

// TerminalHandler 处理终端请求
type TerminalHandler struct {
	k8sClient *k8s.Client
	upgrader  websocket.Upgrader
}

// NewTerminalHandler 创建终端处理器
func NewTerminalHandler(k8sClient *k8s.Client) *TerminalHandler {
	return &TerminalHandler{
		k8sClient: k8sClient,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // 允许所有来源，生产环境应该限制
			},
		},
	}
}

// HandleTerminal 处理终端 WebSocket 连接
func (h *TerminalHandler) HandleTerminal(c *gin.Context) {
	// 获取查询参数
	namespace := c.Query("namespace")
	podName := c.Query("podName")
	containerName := c.Query("containerName")

	log.Printf("Received terminal connection request - namespace: %s, pod: %s, container: %s",
		namespace, podName, containerName)

	if namespace == "" || podName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "namespace and podName are required"})
		return
	}

	// 如果没有指定容器名，使用 Pod 的第一个容器
	if containerName == "" {
		pod, err := h.k8sClient.ClientSet.CoreV1().Pods(namespace).Get(c.Request.Context(), podName, metav1.GetOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get pod: %v", err)})
			return
		}
		if len(pod.Spec.Containers) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Pod has no containers"})
			return
		}
		containerName = pod.Spec.Containers[0].Name
	}

	// 升级 HTTP 连接为 WebSocket
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("WebSocket connection established")

	// 创建终端会话
	session := &TerminalSession{
		wsConn:   conn,
		sizeChan: make(chan remotecommand.TerminalSize),
		doneChan: make(chan struct{}),
	}
	defer close(session.doneChan)

	err = h.executeCommand(namespace, podName, containerName, session)
	if err != nil {
		log.Printf("Failed to execute command: %v", err)
		session.writeMessage(TerminalMessage{
			Op:   "stdout",
			Data: fmt.Sprintf("\r\nError: %v\r\n", err),
		})
	}

	//// 启动 WebSocket 读取协程
	//go session.readWebSocket()
	//
	//// 执行远程命令
	//err = h.executeCommand(namespace, podName, containerName, session)
	//if err != nil {
	//	log.Printf("Failed to execute command: %v", err)
	//	session.Close()
	//	return
	//}
}

// 执行远程命令
func (h *TerminalHandler) executeCommand(namespace, podName, containerName string, session *TerminalSession) error {
	log.Printf("Executing command in container - namespace: %s, pod: %s, container: %s",
		namespace, podName, containerName)

	req := h.k8sClient.ClientSet.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec")

	req.VersionedParams(&v1.PodExecOptions{
		Container: containerName,
		Command:   []string{"/bin/sh", "-c", "[ -x /bin/bash ] && exec /bin/bash || exec /bin/sh"},
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
	}, scheme.ParameterCodec)
	log.Printf("Creating SPDY executor")

	exec, err := remotecommand.NewSPDYExecutor(h.k8sClient.Config, "POST", req.URL())
	if err != nil {
		log.Printf("Failed to create SPDY executor: %v", err)
		return err
	}

	log.Printf("Starting stream")
	// 启动远程命令
	return exec.Stream(remotecommand.StreamOptions{
		Stdin:             session,
		Stdout:            session,
		Stderr:            session,
		Tty:               true,
		TerminalSizeQueue: session,
	})
}

// Next 实现 TerminalSizeQueue 接口
func (t *TerminalSession) Next() *remotecommand.TerminalSize {
	select {
	case size := <-t.sizeChan:
		return &size
	case <-t.doneChan:
		return nil
	}
}

// Read 实现 io.Reader 接口
func (t *TerminalSession) Read(p []byte) (int, error) {
	_, message, err := t.wsConn.ReadMessage()
	if err != nil {
		log.Printf("Failed to read WebSocket message: %v", err)
		return 0, err
	}

	log.Printf("Received message from client: %q", string(message))

	var msg TerminalMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("Failed to unmarshal message: %v", err)
		return 0, err
	}

	switch msg.Op {
	case "stdin":
		log.Printf("Processing stdin: %q", msg.Data)
		return copy(p, msg.Data), nil
	case "resize":
		log.Printf("Processing resize: rows=%d, cols=%d", msg.Rows, msg.Cols)
		t.sizeChan <- remotecommand.TerminalSize{
			Width:  msg.Cols,
			Height: msg.Rows,
		}
		return 0, nil
	default:
		log.Printf("Unknown message type: %s", msg.Op)
		return 0, fmt.Errorf("unknown message type: %s", msg.Op)
	}
}

// Write 实现 io.Writer 接口
func (t *TerminalSession) Write(p []byte) (int, error) {
	log.Printf("Writing to terminal (hex): %x", p)
	log.Printf("Writing to terminal (string): %q", string(p))
	msg := TerminalMessage{
		Op:   "stdout",
		Data: string(p),
	}

	//log.Printf("Writing message - length: %d", len(p))

	if err := t.writeMessage(msg); err != nil {
		log.Printf("Failed to write message: %v", err)
		return 0, err
	}
	return len(p), nil
}

// writeMessage 发送消息到 WebSocket
func (t *TerminalSession) writeMessage(msg TerminalMessage) error {
	return t.wsConn.WriteJSON(msg)
}

// Close 关闭会话
func (t *TerminalSession) Close() {
	close(t.doneChan)
}

//// 读取 WebSocket 消息
//func (t *TerminalSession) readWebSocket() {
//	defer t.Close()
//
//	for {
//		_, _, err := t.wsConn.ReadMessage()
//		if err != nil {
//			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
//				log.Printf("WebSocket error: %v", err)
//			}
//			break
//		}
//	}
//}
