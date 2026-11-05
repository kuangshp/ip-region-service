package public

import (
	ipapi "github.com/box-im/ip-region-service/internal/api/public/location/ip"
	"github.com/box-im/ip-region-service/router/public/location"
	"github.com/gin-gonic/gin"
)

func Register(group *gin.RouterGroup, controller ipapi.IIP) { location.Register(group, controller) }
