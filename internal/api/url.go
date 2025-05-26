package api

import (
	"crypto/rand" // 考虑使用 crypto/rand 获取更强的随机性
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

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errResponse(err))
		return
	}

	// 固定过期时间，可以考虑从请求或配置中获取
	expireDuration := time.Hour * time.Duration(*req.Duration)
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
			// （可选）如果数据库中存在但缓存没有，可以考虑将数据库信息写入缓存
			// fetchedURL, fetchErr := server.store.GetUrlByShortCode(ctx, req.CustomCode) // 假设有这样一个方法
			// if fetchErr == nil {
			// 	ttl := time.Until(fetchedURL.ExpiredAt)
			// 	if ttl > 0 {
			// 		 e := server.rdb.Set(ctx, redisKey, fetchedURL.OriginalUrl, ttl).Err()
			//     if e != nil {
			//        log.Printf("WARN: Failed to update cache for existing custom code '%s': %v\n", req.CustomCode, e)
			//     }
			// 	}
			// }
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
			ShortCode: createdUrl.ShortCode, // 确保响应中包含 OriginalURL
			ExpireAt:  createdUrl.ExpiredAt,
		})
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
			// 同时检查一下 Redis，虽然理论上新生成的应该不会在 Redis 里，除非有哈希碰撞且正好被用作自定义码
			// 这一步可以根据实际情况决定是否需要，主要是防止极小概率的碰撞
			// _, redisErr := server.rdb.Get(ctx, shortcode_prefix+shortCode).Result()
			// if redisErr == redis.Nil {
			// 	return shortCode, nil
			// } else if redisErr != nil {
			//   log.Printf("WARN: Redis check during getUniqueShortCode for '%s' failed: %v\n", shortCode, redisErr)
			//   // 可以选择忽略 Redis 错误继续，或返回错误
			// }
			return shortCode, nil // 如果不检查 Redis，则直接返回
		}
	}
	return "", errors.New("无法在限定次数内生成唯一的短链接")
}

// errResponse 是你自定义的错误响应函数，这里只是一个占位符
func errResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}
