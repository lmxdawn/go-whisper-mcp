package main

import (
	"flag"
	"github.com/sirupsen/logrus"
)

func main() {
	var (
		flagModelsDir string
		flagDefaultM  string // 浏览器二进制文件路径
	)
	flag.StringVar(&flagModelsDir, "models", "./models", "models dir")
	flag.StringVar(&flagDefaultM, "default-model", "medium", "default model spec")
	flag.Parse()

	// 初始化服务
	whisperService := NewWhisperService()

	// 创建并启动应用服务器
	appServer := NewAppServer(flagModelsDir, flagDefaultM, whisperService)
	if err := appServer.Start(":14562"); err != nil {
		logrus.Fatalf("failed to run server: %v", err)
	}
}
