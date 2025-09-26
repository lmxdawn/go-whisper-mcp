package main

import (
	"flag"
	"github.com/sirupsen/logrus"
	"os"
)

func main() {
	var (
		flagDefaultM string // 浏览器二进制文件路径
	)
	flag.StringVar(&flagDefaultM, "default-model", "medium", "default model spec")
	flag.Parse()

	// 初始化服务
	whisperService := NewWhisperService()

	modelsDir := "./models"
	if s := os.Getenv("MODELS_DIR"); len(s) > 0 {
		modelsDir = s
	}

	// 创建并启动应用服务器
	appServer := NewAppServer(modelsDir, flagDefaultM, whisperService)
	if err := appServer.Start(":28796"); err != nil {
		logrus.Fatalf("failed to run server: %v", err)
	}
}
