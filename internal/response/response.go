package response

import "github.com/gin-gonic/gin"

type Envelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func Write(ctx *gin.Context, status, code int, message string, data any) {
	ctx.JSON(status, Envelope{Code: code, Message: message, Data: data})
}
