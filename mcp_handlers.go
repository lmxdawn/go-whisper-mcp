package main

import (
	"bytes"
	"context"
	"encoding/json"
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
		Threads:   t,
		ModelsDir: a.modelsDir,
	}

	transcribeBatchResponse, err := a.whisperService.Transcribe(ctx, req)
	if err != nil {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "转换失败: " + err.Error()}}, IsError: true}
	}

	var list []string

	for _, result := range transcribeBatchResponse.Results {
		var buffer bytes.Buffer
		for _, segment := range result.Segments {
			buffer.WriteString(segment.Text)
		}
		list = append(list, buffer.String()+",")
	}

	jsonData, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: fmt.Sprintf("转换成功，但序列化失败: %v", err),
			}},
			IsError: true,
		}
	}

	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text", Text: string(jsonData),
		}},
		IsError: false,
	}
}
