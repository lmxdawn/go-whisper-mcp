package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// AppServer 应用服务器结构体，封装所有服务和处理器
type AppServer struct {
	whisperService *WhisperService
	router         *gin.Engine
	httpServer     *http.Server
	modelsDir      string
	defaultModel   string
}

// NewAppServer 创建新的应用服务器实例
func NewAppServer(modelsDir string, defaultModel string, whisperService *WhisperService) *AppServer {
	return &AppServer{
		whisperService: whisperService,
		modelsDir:      modelsDir,
		defaultModel:   defaultModel,
	}
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

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 关闭 HTTP 服务器
	if err := a.httpServer.Shutdown(ctx); err != nil {
		logrus.Errorf("服务器关闭失败: %v", err)
		return err
	}

	logrus.Infof("服务器已关闭")
	return nil
}
