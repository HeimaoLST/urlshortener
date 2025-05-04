package api

import (
	"errors"
	"github/heimaolst/urlshorter/internal/model"
	"net/http"

	"github.com/gin-gonic/gin"
)

// POST 短链接生成
func (server *Server) CreateURL(ctx *gin.Context) {

	var req model.CreateURLRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errResponse(err))
		return
	}
	if req.CustomCode != "" {
		// 自定义短链接
		isAvailable, err := server.store.IsShortCodeAvailable(ctx, req.CustomCode)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, errResponse(err))
			return
		}
		if !isAvailable {
			ctx.JSON(http.StatusBadRequest, errResponse(errors.New("短链接已存在")))
			return
		}

	}

	// 生成短链接

	// shortcode, err := server.store.CreateURL(ctx, arg)
	// if err != nil {
	// 	ctx.JSON(http.StatusInternalServerError, errResponse(err))
	// 	return
	// }

	// res := model.CreateURLResponse{
	// 	ShortURL: shortcode,
	// 	ExpireAt: shortcode.ExpireAt,
	// }
	// ctx.JSON(http.StatusOK, res)

}
