package api

import (
	"errors"
	"github/heimaolst/urlshorter/internal/model"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

var shortcode_prefix = "urlshortener:shortcode:"

// POST 短链接生成
func (server *Server) CreateURL(ctx *gin.Context) {

	var req model.CreateURLRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errResponse(err))
		return
	}
	if req.CustomCode != "" {
		// TODO:自定义短链接
		// 1. 检查短链接是否在缓存中
		// 2. 如果存在，返回错误
		// 3. 如果不存在，检查数据库中是否存在
		// 4. 如果存在，返回错误，同时将短链接存入缓存
		// 5. 如果不存在，生成短链接，存入数据库和缓存
		_, err := server.rdb.Get(ctx, shortcode_prefix+req.CustomCode).Result()
		if err == redis.Nil {
			isAvailable, err := server.store.IsShortCodeAvailable(ctx, req.CustomCode)
			if err != nil {
				ctx.JSON(http.StatusInternalServerError, errResponse(err))
				return
			}

			if !isAvailable {
				ctx.JSON(http.StatusBadRequest, errResponse(errors.New("自定义短链接已被使用")))
				expireAt, err := server.store.UrlExpiredAt(ctx, req.CustomCode)
				if err != nil {
					log.Fatal(err)
					return
				}
				ttl := time.Until(expireAt)
				_, err = server.rdb.Set(ctx, shortcode_prefix+req.CustomCode, req.OriginalURL, ttl).Result()
				if err != nil {
					log.Fatal(err)
				}
				return
			}
			//创建新的短链接

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

func generateShortCode() (string, error) {
	rand.Seed(time.Now().UnixNano())
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	length := 6
	shortCode := make([]byte, length)
	for i := range shortCode {
		shortCode[i] = charset[rand.Intn(len(charset))]
	}
	return string(shortCode), nil

}
func getShortCode(ctx *gin.Context, n int) (string, error) {
	if n > 5 {
		return "", errors.New("重试过多")
	}
	code, err := generateShortCode()
	if err != nil {
		return "", err
	}
	_, err = server.rdb.Get(ctx, shortcode_prefix+code).Result()
	if err == redis.Nil {
		// 生成短链接成功
		return code, nil
	}
	

}
