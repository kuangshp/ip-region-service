package ip

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/box-im/ip-region-service/internal/api/public/location/ip/dto"
	"github.com/box-im/ip-region-service/internal/api/public/location/ip/vo"
	"github.com/box-im/ip-region-service/internal/response"
	"github.com/box-im/ip-region-service/pkg/ipregion"
	"github.com/gin-gonic/gin"
)

type LookupService interface {
	Lookup(string) (ipregion.Region, error)
}
type IIP interface{ Lookup(*gin.Context) }
type controller struct{ lookup LookupService }

var _ IIP = (*controller)(nil)

func NewIP(lookup LookupService) IIP { return &controller{lookup: lookup} }

func (c *controller) Lookup(ctx *gin.Context) {
	var request dto.LookupRequest
	var err error
	if ctx.Request.Method == http.MethodGet {
		err = ctx.ShouldBindQuery(&request)
	} else {
		ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 4096)
		if ctx.ContentType() != "application/json" {
			response.Write(ctx, http.StatusUnsupportedMediaType, 415, "请使用 application/json", nil)
			return
		}
		err = ctx.ShouldBindJSON(&request)
	}
	if err != nil {
		response.Write(ctx, http.StatusBadRequest, 400, "请传入有效的 ip 参数", nil)
		return
	}
	region, err := c.lookup.Lookup(request.IP)
	switch {
	case errors.Is(err, ipregion.ErrInvalidIP):
		response.Write(ctx, http.StatusBadRequest, 400, "IP 地址格式不正确", nil)
	case errors.Is(err, ipregion.ErrNotFound):
		response.Write(ctx, http.StatusNotFound, 404, "未查询到 IP 归属地", nil)
	case errors.Is(err, ipregion.ErrUnsupported):
		response.Write(ctx, http.StatusUnprocessableEntity, 422, "未配置此 IP 版本的数据文件", nil)
	case err != nil:
		slog.Error("IP lookup failed", "error", err)
		response.Write(ctx, http.StatusInternalServerError, 500, "IP 归属地查询失败", nil)
	default:
		response.Write(ctx, http.StatusOK, 0, "success", vo.LookupResponse{Region: region})
	}
}
