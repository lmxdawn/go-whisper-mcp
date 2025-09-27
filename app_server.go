package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sirupsen/logrus"
)

// AppServer 应用服务器结构体，封装所有服务和处理器
type AppServer struct {
	whisperService *WhisperService
	mcpServer      *mcp.Server
	router         *gin.Engine
	httpServer     *http.Server
	modelsDir      string
	defaultModel   string
}

// NewAppServer 创建新的应用服务器实例
func NewAppServer(modelsDir string, defaultModel string, whisperService *WhisperService) *AppServer {
	appServer := &AppServer{
		whisperService: whisperService,
		modelsDir:      modelsDir,
		defaultModel:   defaultModel,
	}

	// 初始化 MCP Server（需要在创建 appServer 之后，因为工具注册需要访问 appServer）
	appServer.mcpServer = InitMCPServer(appServer)

	return appServer
}

// Start 启动服务器
func (a *AppServer) Start(port string) error {
	a.router = setupRoutes(a)

	a.httpServer = &http.Server{
		Addr:    port,
		Handler: a.router,
	}

	// 启动服务器的 goroutine
	go func() {
		logrus.Infof("启动 HTTP 服务器: %s", port)
		if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Errorf("服务器启动失败: %v", err)
			os.Exit(1)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logrus.Infof("正在关闭服务器...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := a.httpServer.Shutdown(ctx); err != nil {
		logrus.Warnf("等待连接关闭超时，强制退出: %v", err)
	} else {
		logrus.Infof("服务器已优雅关闭")
	}

	logrus.Infof("服务器已关闭")
	return nil
}
