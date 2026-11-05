package initialize

import (
	"net/http"

	ipapi "github.com/box-im/ip-region-service/internal/api/public/location/ip"
	"github.com/box-im/ip-region-service/internal/response"
	"github.com/box-im/ip-region-service/router/public"
	"github.com/gin-gonic/gin"
)

func Router(lookup ipapi.LookupService) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Logger(), gin.CustomRecovery(func(ctx *gin.Context, _ any) {
		response.Write(ctx, 500, 500, "服务内部错误", nil)
		ctx.Abort()
	}))
	// Public read-only service: allow browser calls without cookies or credentials.
	engine.Use(func(ctx *gin.Context) {
		ctx.Header("Access-Control-Allow-Origin", "*")
		ctx.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		ctx.Header("Access-Control-Allow-Headers", "Content-Type")
		if ctx.Request.Method == http.MethodOptions {
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}
		ctx.Next()
	})
	engine.GET("/healthz", func(ctx *gin.Context) { response.Write(ctx, 200, 0, "ok", nil) })
	public.Register(engine.Group("/api/v1"), ipapi.NewIP(lookup))
	engine.NoRoute(func(ctx *gin.Context) { response.Write(ctx, 404, 404, "接口不存在", nil) })
	return engine
}
