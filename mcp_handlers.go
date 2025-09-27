package main

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
)

// MCP 工具处理函数

// handleTranscribe 转换
func (a *AppServer) handleTranscribe(ctx context.Context, args map[string]interface{}) *MCPToolResult {
	logrus.Info("MCP: 转换", args)

	inPaths, _ := args["in_paths"].([]interface{})
	model, _ := args["model"].(string)
	lang, _ := args["lang"].(string)
	t, _ := args["t"].(int)

	var mediaPaths []string
	for _, path := range inPaths {
		if pathStr, ok := path.(string); ok {
			mediaPaths = append(mediaPaths, pathStr)
		}
	}

	if len(model) == 0 {
		model = a.defaultModel
	}

	req := &TranscribeRequest{
		InPaths:   mediaPaths,
		Model:     model,
		Lang:      lang,
		Threads:   int(t),
		ModelsDir: a.modelsDir,
	}

	transcribeBatchResponse, err := a.whisperService.Transcribe(ctx, req)
	if err != nil {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "转换失败: " + err.Error()}}, IsError: true}
	}

	// 生成摘要
	resultCount := len(transcribeBatchResponse.Results)
	summary := fmt.Sprintf("ok (%d results, %.2fs total)", resultCount, transcribeBatchResponse.DurationS)

	return &MCPToolResult{
		Content:           []MCPContent{{Type: "text", Text: summary}},
		StructuredContent: transcribeBatchResponse, // 直接塞结构体
		IsError:           false,
	}
}
