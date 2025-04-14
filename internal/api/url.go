package api

import (
	"context"
	"github/heimaolst/urlshorter/internal/model"
	"net/http"

	"github.com/labstack/echo/v4"
)

type URLHandler struct {
	urlService URLService
}
type URLService interface {
	CreateURL(ctx context.Context, req model.CreateURLRequest) (*model.CreateURLResponse, error)
}

// POST 短链接生成
func (h *URLHandler) CreateURL(c echo.Context) error {

	var req model.CreateURLRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	resp, err := h.urlService.CreateURL(c.Request().Context(), req)

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusCreated, resp)

}

//GET 重定向到长链接
