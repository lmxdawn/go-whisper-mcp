package main

import (
	"context"
	"encoding/base64"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sirupsen/logrus"
)

// MCP 工具参数结构体定义

// TranscribeArgs 发布内容的参数
type TranscribeArgs struct {
	InPaths []string `json:"in_paths" jsonschema:"mp4 或 wav 的本地文件路径"`
	Model   string   `json:"model" jsonschema:"模型规格或文件名（例如 tiny、medium、large-v3、ggml-small.bin）"`
	Lang    string   `json:"lang" jsonschema:"语言代码或“auto”（例如 zh、en、auto）"`
	Threads int      `json:"t" jsonschema:"线程"`
}

// InitMCPServer 初始化 MCP Server
func InitMCPServer(appServer *AppServer) *mcp.Server {
	// 创建 MCP Server
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "go-whisper-mcp",
			Version: "2.0.0",
		},
		nil,
	)

	// 注册所有工具
	registerTools(server, appServer)

	logrus.Info("MCP Server initialized with official SDK")

	return server
}

// registerTools 注册所有 MCP 工具
func registerTools(server *mcp.Server, appServer *AppServer) {
	// 工具 1: 转换
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "transcribe",
			Description: "将 mp4/wav 转录为文本（支持 model/lang/threads）",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args TranscribeArgs) (*mcp.CallToolResult, any, error) {
			argsMap := map[string]any{
				"in_paths": convertStringsToInterfaces(args.InPaths),
				"model":    args.Model,
				"lang":     args.Lang,
				"t":        args.Threads,
			}
			r := appServer.handleTranscribe(ctx, argsMap) // *MCPToolResult

			// 1) 内容区：摘要文本 / 图片（不塞 structured）
			res := convertToMCPResult(r)

			return res, nil, nil
		},
	)

	logrus.Infof("Registered %d MCP tools", 8)
}

// convertToMCPResult 将自定义的 MCPToolResult 转换为官方 SDK 的格式
func convertToMCPResult(result *MCPToolResult) *mcp.CallToolResult {
	var contents []mcp.Content
	for _, c := range result.Content {
		switch c.Type {
		case "text":
			contents = append(contents, &mcp.TextContent{Text: c.Text})
		case "image":
			// 解码 base64 字符串为 []byte
			imageData, err := base64.StdEncoding.DecodeString(c.Data)
			if err != nil {
				logrus.WithError(err).Error("Failed to decode base64 image data")
				// 如果解码失败，添加错误文本
				contents = append(contents, &mcp.TextContent{
					Text: "图片数据解码失败: " + err.Error(),
				})
			} else {
				contents = append(contents, &mcp.ImageContent{
					Data:     imageData,
					MIMEType: c.MimeType,
				})
			}
		}
	}

	return &mcp.CallToolResult{
		StructuredContent: result.StructuredContent,
		Content:           contents,
		IsError:           result.IsError,
	}
}

// convertStringsToInterfaces 辅助函数：将 []string 转换为 []interface{}
func convertStringsToInterfaces(strs []string) []interface{} {
	result := make([]interface{}, len(strs))
	for i, s := range strs {
		result[i] = s
	}
	return result
}
