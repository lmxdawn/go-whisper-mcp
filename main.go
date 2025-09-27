package main

import (
	"flag"
	"os"

	"github.com/sirupsen/logrus"
)

func main() {
	if os.Getenv("NATIVE_LOG_SILENT") != "0" { // 缺省静音；设置 0 则打开
		DisableNativeLogs()
	}

	var (
		flagDefaultM string // 浏览器二进制文件路径
		port         string
	)
	flag.StringVar(&flagDefaultM, "default-model", "medium", "default model spec")
	flag.StringVar(&port, "port", ":28796", "端口")
	flag.Parse()

	// 初始化服务
	whisperService := NewWhisperService()

	modelsDir := "./models"
	if s := os.Getenv("MODELS_DIR"); len(s) > 0 {
		modelsDir = s
	}

	// 创建并启动应用服务器
	appServer := NewAppServer(modelsDir, flagDefaultM, whisperService)
	if err := appServer.Start(port); err != nil {
		logrus.Fatalf("failed to run server: %v", err)
	}
}
