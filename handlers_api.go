package main

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
)

// respondError 返回错误响应
func respondError(c *gin.Context, statusCode int, code, message string, details any) {
	response := ErrorResponse{
		Error:   message,
		Code:    code,
		Details: details,
	}

	logrus.Errorf("%s %s %s %d", c.Request.Method, c.Request.URL.Path,
		c.GetString("account"), statusCode)

	c.JSON(statusCode, response)
}

// respondSuccess 返回成功响应
func respondSuccess(c *gin.Context, data any, message string) {
	response := SuccessResponse{
		Success: true,
		Data:    data,
		Message: message,
	}

	logrus.Infof("%s %s %s %d", c.Request.Method, c.Request.URL.Path,
		c.GetString("account"), http.StatusOK)

	c.JSON(http.StatusOK, response)
}

func handleTranscribe(a *AppServer) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req TranscribeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			respondError(c, http.StatusBadRequest, "INVALID_REQUEST",
				"请求参数错误", err.Error())
			return
		}

		if len(req.Model) == 0 {
			req.Model = a.defaultModel
		}

		out, err := a.whisperService.Transcribe(c.Request.Context(), &req)
		if err != nil {
			respondError(c, http.StatusInternalServerError, "TranscribeError", "transcription failed", err.Error())
			return
		}

		respondSuccess(c, out, "ok")
	}
}

// healthHandler 健康检查
func healthHandler(c *gin.Context) {
	respondSuccess(c, map[string]any{
		"status":    "healthy",
		"service":   "go-whisper-mcp",
		"account":   "ai-report",
		"timestamp": "now",
	}, "服务正常")
}
