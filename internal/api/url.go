package api

import (
	"crypto/rand" // 考虑使用 crypto/rand 获取更强的随机性
	"database/sql"
	"errors"
	"fmt"
	db "github/heimaolst/urlshorter/db/sqlc"
	"github/heimaolst/urlshorter/internal/model"

	"log"
	"math/big" // 如果使用 crypto/rand
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

var shortcode_prefix = "urlshortener:shortcode:"

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const defaultShortCodeLength = 6
const maxGenerateRetries = 5

// Server struct 可能包含 rand.Rand 实例
// type Server struct {
// 	rdb  *redis.Client
// 	store db.Store // 假设 store 是你的数据库操作接口
// 	// randGen *mathrand.Rand // math/rand.Rand
// }

// newRand := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
// server := &Server{randGen: newRand, ...}

// POST 短链接生成
func (server *Server) CreateURL(ctx *gin.Context) {

	var req model.CreateURLRequest
	var expireDuration time.Duration

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errResponse(err))
		return
	}
	log.Println(">>>>>>request:", req)

	// 固定过期时间，可以考虑从请求或配置中获取
	if req.Duration != nil {
		// 只有当 req.Duration 不是 nil 时才解引用
		expireDuration = time.Hour * time.Duration(*req.Duration)
	} else {
		expireDuration = time.Hour * 1
	}
	finalExpireAt := time.Now().Add(expireDuration)

	if req.CustomCode != "" {
		// --- 处理自定义短链接 ---
		redisKey := shortcode_prefix + req.CustomCode

		// 1. 检查缓存
		_, err := server.rdb.Get(ctx, redisKey).Result()
		if err == nil { // 缓存命中
			ctx.JSON(http.StatusConflict, errResponse(errors.New("自定义短链接已被使用 (缓存)")))
			return
		}
		if err != redis.Nil { // Redis 查询出错
			log.Printf("ERROR: Redis Get for custom code '%s' failed: %v\n", req.CustomCode, err)
			ctx.JSON(http.StatusInternalServerError, errResponse(errors.New("服务内部错误，请稍后重试")))
			return
		}

		// 缓存未命中 (err == redis.Nil)，检查数据库
		isAvailable, dbErr := server.store.IsShortCodeAvailable(ctx, req.CustomCode) // 假设此方法返回 true 如果可用，false 如果已存在
		if dbErr != nil {
			log.Printf("ERROR: DB IsShortCodeAvailable for custom code '%s' failed: %v\n", req.CustomCode, dbErr)
			ctx.JSON(http.StatusInternalServerError, errResponse(errors.New("服务内部错误，请稍后重试")))
			return
		}

		if !isAvailable {

			ctx.JSON(http.StatusConflict, errResponse(errors.New("自定义短链接已被使用 (数据库)")))
			return
		}

		// 自定义短链接可用，创建
		urlParams := db.CreateUrlParams{
			ShortCode:   req.CustomCode,
			IsCustom:    true,
			OriginalUrl: req.OriginalURL,
			ExpiredAt:   finalExpireAt,
		}
		createdUrl, err := server.store.CreateUrl(ctx, urlParams) // 假设 CreateUrl 返回创建的对象
		if err != nil {
			log.Printf("ERROR: DB CreateUrl for custom code '%s' failed: %v\n", req.CustomCode, err)
			ctx.JSON(http.StatusInternalServerError, errResponse(errors.New("创建短链接失败")))
			return
		}

		ttl := time.Until(createdUrl.ExpiredAt)
		if ttl > 0 {
			err = server.rdb.Set(ctx, redisKey, createdUrl.OriginalUrl, ttl).Err()
			if err != nil {
				log.Printf("WARN: Redis Set for custom code '%s' failed after DB insert: %v\n", req.CustomCode, err)
				// 通常不因为缓存失败而给用户报错，数据库已成功
			}
		}

		ctx.JSON(http.StatusOK, model.CreateURLResponse{
			Success:   true,
			ShortCode: createdUrl.ShortCode, // 确保响应中包含 OriginalURL
			ExpireAt:  createdUrl.ExpiredAt,
		})
		return

	} else {
		// --- 处理自动生成短链接 ---
		shortCode, err := server.getUniqueShortCode(ctx)
		if err != nil {
			log.Printf("ERROR: Failed to generate unique short code: %v\n", err)
			ctx.JSON(http.StatusInternalServerError, errResponse(errors.New("生成短链接失败")))
			return
		}

		urlParams := db.CreateUrlParams{
			ShortCode:   shortCode,
			IsCustom:    false,
			OriginalUrl: req.OriginalURL,
			ExpiredAt:   finalExpireAt,
		}
		createdUrl, err := server.store.CreateUrl(ctx, urlParams)
		if err != nil {
			log.Printf("ERROR: DB CreateUrl for generated code '%s' failed: %v\n", shortCode, err)
			ctx.JSON(http.StatusInternalServerError, errResponse(errors.New("创建短链接失败")))
			return
		}

		redisKey := shortcode_prefix + shortCode
		ttl := time.Until(createdUrl.ExpiredAt)
		if ttl > 0 {
			err = server.rdb.Set(ctx, redisKey, createdUrl.OriginalUrl, ttl).Err()
			if err != nil {
				log.Printf("WARN: Redis Set for generated code '%s' failed after DB insert: %v\n", shortCode, err)
			}
		}

		ctx.JSON(http.StatusOK, model.CreateURLResponse{
			Success:   true,
			ShortCode: createdUrl.ShortCode, // 确保响应中包含 OriginalURL
			ExpireAt:  createdUrl.ExpiredAt,
		})
		return
	}
}
func (server *Server) RedirectURL(ctx *gin.Context) {
	shortcode := ctx.Query("shortcode")
	if shortcode == "" {
		ctx.JSON(http.StatusBadRequest, errResponse(errors.New("缺少 shortcode 参数")))
		return
	}
	// 先检查 Redis 缓存
	redisKey := shortcode_prefix + shortcode
	originalURL, err := server.rdb.Get(ctx, redisKey).Result()
	if err == redis.Nil {
		// 缓存未命中，查询数据库
		url, dbErr := server.store.GetUrlByShortCode(ctx, shortcode)
		if dbErr != nil {
			if dbErr == sql.ErrNoRows {
				ctx.JSON(http.StatusNotFound, errResponse(errors.New("短链接不存在")))
			} else {
				log.Printf("ERROR: DB GetUrlByShortCode for '%s' failed: %v\n", shortcode, dbErr)
				ctx.JSON(http.StatusInternalServerError, errResponse(errors.New("服务内部错误，请稍后重试")))
			}
			return
		}

		// 检查是否过期
		if time.Now().After(url.ExpiredAt) {
			ctx.JSON(http.StatusGone, errResponse(errors.New("短链接已过期")))
			return
		}

		// 将结果存入 Redis 缓存
		err = server.rdb.Set(ctx, redisKey, url.OriginalUrl, time.Until(url.ExpiredAt)).Err()
		if err != nil {
			log.Printf("WARN: Redis Set for '%s' after DB fetch failed: %v\n", shortcode, err)
			// 通常不因为缓存失败而给用户报错，数据库已成功
		}

		ctx.Redirect(http.StatusFound, url.OriginalUrl)
		return
	} else if err != nil {
		log.Printf("ERROR: Redis Get for '%s' failed: %v\n", shortcode, err)
		ctx.JSON(http.StatusInternalServerError, errResponse(errors.New("服务内部错误，请稍后重试")))
		return

	} else {
		ctx.Redirect(http.StatusFound, originalURL)
		return
	}

}

// generateRandomString 生成指定长度的随机字符串
// 建议: 将此函数或其使用的 rand.Rand 实例作为 Server 的一部分，以便更好地管理随机数生成器的状态和播种。
func generateRandomString(length int) (string, error) {
	// 使用 crypto/rand
	sb := make([]byte, length)
	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", fmt.Errorf("failed to generate random char index: %w", err)
		}
		sb[i] = charset[num.Int64()]
	}
	return string(sb), nil

	// 如果使用 math/rand (确保已在某处播种，例如 server.randGen.Intn)
	// shortCode := make([]byte, length)
	// for i := range shortCode {
	// 	shortCode[i] = charset[globalOrServerRand.Intn(len(charset))]
	// }
	// return string(shortCode), nil
}

func (server *Server) getUniqueShortCode(ctx *gin.Context) (string, error) {
	var shortCode string
	var err error
	for i := 0; i < maxGenerateRetries; i++ {
		shortCode, err = generateRandomString(defaultShortCodeLength) // 或你原来的 generateShortCode
		if err != nil {
			return "", fmt.Errorf("failed to generate initial short code: %w", err)
		}

		isAvailable, dbErr := server.store.IsShortCodeAvailable(ctx, shortCode)
		if dbErr != nil {
			return "", fmt.Errorf("db check for IsShortCodeAvailable failed for '%s': %w", shortCode, dbErr)
		}

		if isAvailable {

			_, redisErr := server.rdb.Get(ctx, shortcode_prefix+shortCode).Result()
			if redisErr == redis.Nil {
				return shortCode, nil
			} else if redisErr != nil {
				log.Printf("WARN: Redis check during getUniqueShortCode for '%s' failed: %v\n", shortCode, redisErr)
			}

		}
	}
	return "", errors.New("无法在限定次数内生成唯一的短链接")
}

// errResponse 是你自定义的错误响应函数，这里只是一个占位符
func errResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}
