package utils

import "github.com/gin-gonic/gin"

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
	Meta    any    `json:"meta,omitempty"`
}

func Success(c *gin.Context, status int, message string, data any) {
	c.JSON(status, Response{Success: true, Message: message, Data: data})
}

func SuccessWithMeta(c *gin.Context, status int, message string, data any, meta any) {
	c.JSON(status, Response{Success: true, Message: message, Data: data, Meta: meta})
}

func Error(c *gin.Context, status int, message string) {
	c.JSON(status, Response{Success: false, Error: message})
}
