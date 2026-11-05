package location

import (
	ipapi "github.com/box-im/ip-region-service/internal/api/public/location/ip"
	"github.com/gin-gonic/gin"
)

func Register(group *gin.RouterGroup, controller ipapi.IIP) { registerIP(group, controller) }
