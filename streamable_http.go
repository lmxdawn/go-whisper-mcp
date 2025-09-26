package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

// StreamableHTTPHandler 处理 Streamable HTTP 协议的 MCP 请求
func (a *AppServer) StreamableHTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 设置 CORS 头
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, Mcp-Session-Id")

		// 处理 OPTIONS 请求
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// 根据方法处理
		switch r.Method {
		case "GET":
			// GET 请求用于建立 SSE 连接（可选功能）
			a.handleSSEConnection(w, r)
		case "POST":
			// POST 请求处理 JSON-RPC
			a.handleJSONRPCRequest(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// handleSSEConnection 处理 SSE 连接（可选，用于服务器推送）
func (a *AppServer) handleSSEConnection(w http.ResponseWriter, r *http.Request) {
	// 检查是否支持 SSE
	if !strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
		http.Error(w, "SSE not requested", http.StatusBadRequest)
		return
	}

	// 设置 SSE 响应头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// 发送初始化消息
	fmt.Fprintf(w, "event: open\n")
	fmt.Fprintf(w, "data: {\"type\":\"connection\",\"status\":\"connected\"}\n\n")

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// 保持连接打开（实际使用中可以在这里推送通知）
	<-r.Context().Done()
}

// handleJSONRPCRequest 处理 JSON-RPC 请求
func (a *AppServer) handleJSONRPCRequest(w http.ResponseWriter, r *http.Request) {
	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		a.sendStreamableError(w, nil, -32700, "Parse error")
		return
	}
	defer r.Body.Close()

	// 解析 JSON-RPC 请求
	var request JSONRPCRequest
	if err := json.Unmarshal(body, &request); err != nil {
		a.sendStreamableError(w, nil, -32700, "Parse error")
		return
	}

	logrus.WithField("method", request.Method).Info("Received Streamable HTTP request")

	// 检查 Accept 头，判断客户端是否支持 SSE
	acceptSSE := strings.Contains(r.Header.Get("Accept"), "text/event-stream")

	// 处理请求
	response := a.processJSONRPCRequest(&request, r.Context())

	// 如果需要 SSE 且是支持流式的方法，使用 SSE 响应
	if acceptSSE && a.isStreamableMethod(request.Method) {
		a.sendSSEResponse(w, response)
	} else {
		// 否则使用普通 JSON 响应
		a.sendJSONResponse(w, response)
	}
}

// processJSONRPCRequest 处理 JSON-RPC 请求并返回响应
func (a *AppServer) processJSONRPCRequest(request *JSONRPCRequest, ctx context.Context) *JSONRPCResponse {
	switch request.Method {
	case "initialize":
		return a.processInitialize(request)
	case "initialized":
		// 客户端确认初始化完成
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			Result:  map[string]interface{}{},
			ID:      request.ID,
		}
	case "ping":
		// 处理 ping 请求
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			Result:  map[string]interface{}{},
			ID:      request.ID,
		}
	case "tools/list":
		return a.processToolsList(request)
	case "tools/call":
		return a.processToolCall(ctx, request)
	default:
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &JSONRPCError{
				Code:    -32601,
				Message: "Method not found",
			},
			ID: request.ID,
		}
	}
}

// processInitialize 处理初始化请求
func (a *AppServer) processInitialize(request *JSONRPCRequest) *JSONRPCResponse {
	result := map[string]interface{}{
		"protocolVersion": "2025-03-26", // 使用新的协议版本
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "go-whisper-mcp",
			"version": "2.0.0",
		},
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      request.ID,
	}
}

// processToolsList 处理工具列表请求
func (a *AppServer) processToolsList(request *JSONRPCRequest) *JSONRPCResponse {
	tools := []map[string]interface{}{
		{
			"name":        "transcribe",
			"description": "Transcribe mp4/wav to text using whisper.cpp (supports model/lang/threads).",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"in_paths": map[string]any{
						"type":        "array",
						"description": "Local file path to mp4 or wav.",
						"items": map[string]interface{}{
							"type": "string",
						},
						"minItems": 1,
					},
					"model": map[string]any{
						"type":        "string",
						"description": "Model spec or filename (e.g. tiny, medium, large-v3, ggml-small.bin).",
					},
					"lang": map[string]any{
						"type": "string", "description": "Language code or 'auto' (e.g. zh, en, auto).",
					},
					"t": map[string]any{
						"type": "integer", "description": "Threads", "minimum": 1,
					},
				},
				"required": []string{"in_paths", "t"},
			},
		},
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		Result: map[string]interface{}{
			"tools": tools,
		},
		ID: request.ID,
	}
}

// processToolCall 处理工具调用
func (a *AppServer) processToolCall(ctx context.Context, request *JSONRPCRequest) *JSONRPCResponse {
	// 解析参数
	params, ok := request.Params.(map[string]interface{})
	if !ok {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &JSONRPCError{
				Code:    -32602,
				Message: "Invalid params",
			},
			ID: request.ID,
		}
	}

	toolName, _ := params["name"].(string)
	toolArgs, _ := params["arguments"].(map[string]interface{})

	var result *MCPToolResult

	switch toolName {
	case "transcribe":
		result = a.handleTranscribe(ctx, toolArgs)
	default:
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &JSONRPCError{
				Code:    -32602,
				Message: fmt.Sprintf("Unknown tool: %s", toolName),
			},
			ID: request.ID,
		}
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      request.ID,
	}
}

// isStreamableMethod 判断方法是否支持流式响应
func (a *AppServer) isStreamableMethod(_ string) bool {
	// 目前我们的方法都不需要流式响应
	// 未来可以在这里添加支持流式的方法
	return false
}

// sendJSONResponse 发送普通 JSON 响应
func (a *AppServer) sendJSONResponse(w http.ResponseWriter, response *JSONRPCResponse) {
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logrus.WithError(err).Error("Failed to encode response")
	}
}

// sendSSEResponse 发送 SSE 响应
func (a *AppServer) sendSSEResponse(w http.ResponseWriter, response *JSONRPCResponse) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// 将响应转换为 JSON
	data, err := json.Marshal(response)
	if err != nil {
		logrus.WithError(err).Error("Failed to marshal SSE response")
		return
	}

	// 发送 SSE 格式的响应
	fmt.Fprintf(w, "data: %s\n\n", string(data))

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// sendStreamableError 发送错误响应
func (a *AppServer) sendStreamableError(w http.ResponseWriter, id interface{}, code int, message string) {
	response := &JSONRPCResponse{
		JSONRPC: "2.0",
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
		},
		ID: id,
	}
	a.sendJSONResponse(w, response)
}
