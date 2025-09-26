package main

import "github.com/gin-gonic/gin"

func setupRoutes(a *AppServer) *gin.Engine {
	// 设置模式
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.Use(errorHandlingMiddleware())
	r.Use(corsMiddleware())

	// 健康检查
	r.GET("/health", healthHandler)

	// MCP over HTTP
	mcpHandler := a.StreamableHTTPHandler()
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
