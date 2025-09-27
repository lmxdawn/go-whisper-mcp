package main

// HTTP API 响应类型

// ErrorResponse 错误响应
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Details any    `json:"details,omitempty"`
}

// SuccessResponse 成功响应
type SuccessResponse struct {
	Success bool   `json:"success"`
	Data    any    `json:"data"`
	Message string `json:"message,omitempty"`
}

// MCPToolResult MCP 工具结果
type MCPToolResult struct {
	Content           []MCPContent `json:"content"`
	StructuredContent any          `json:"structuredContent,omitempty"`
	IsError           bool         `json:"isError,omitempty"`
}

// MCPContent MCP 内容
type MCPContent struct {
	Type     string `json:"type"`
	Text     string `json:"text"`
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}
