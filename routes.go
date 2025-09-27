package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func setupRoutes(a *AppServer) *gin.Engine {
	// 设置模式
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.Use(errorHandlingMiddleware())
	r.Use(corsMiddleware())

	// 健康检查
	r.GET("/health", healthHandler)

	// MCP 端点 - 使用官方 SDK 的 Streamable HTTP Handler
	mcpHandler := mcp.NewStreamableHTTPHandler(
		func(r *http.Request) *mcp.Server {
			return a.mcpServer
		},
		&mcp.StreamableHTTPOptions{
			JSONResponse: true, // 支持 JSON 响应
		},
	)
	r.Any("/mcp", gin.WrapH(mcpHandler))
	r.Any("/mcp/*path", gin.WrapH(mcpHandler))

	// REST 组
	rest := r.Group("/api")
	{
		// 业务 API
		rest.POST("/transcribe", handleTranscribe(a))
	}

	return r
}
