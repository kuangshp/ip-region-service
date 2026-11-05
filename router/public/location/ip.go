package location

import (
	ipapi "github.com/box-im/ip-region-service/internal/api/public/location/ip"
	"github.com/gin-gonic/gin"
)

func registerIP(group *gin.RouterGroup, controller ipapi.IIP) {
	group.GET("/ip", controller.Lookup)
	group.POST("/ip", controller.Lookup)
}
