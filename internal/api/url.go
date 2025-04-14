package api

import (
	"github.com/labstack/echo/v4"
)

type URLHandler struct {
}

// POST 短链接生成
func (h *URLHandler) CreateURL(c echo.Context) error {

}

//GET 重定向到长链接
