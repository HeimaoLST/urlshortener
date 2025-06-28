package api

import (
	"context"
	"crypto/rand" // 考虑使用 crypto/rand 获取更强的随机性
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	db "github/heimaolst/urlshorter/db/sqlc"
	"github/heimaolst/urlshorter/internal/model"
	"github/heimaolst/urlshorter/internal/util"
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

	// 计算最终的过期时间
	var expireDuration time.Duration
	if req.Duration != nil {
		expireDuration = time.Hour * time.Duration(*req.Duration)
	} else {
		expireDuration = time.Hour * 1 // 默认1小时
	}
	finalExpireAt := time.Now().Add(expireDuration)

	var shortCode string
	var isCustom bool

	if req.CustomCode != "" {
		// --- 分支一: 处理自定义短链接 ---
		isCustom = true
		shortCode = req.CustomCode

		// 1. 检查自定义短链接是否已被使用 (统一检查缓存和数据库)
		// 首先检查缓存
		cachedUrl, err := server.getUrlFromCache(ctx, shortCode)
		if err != nil && err != redis.Nil {
			log.Printf("ERROR: Redis check for custom code '%s' failed: %v\n", shortCode, err)

		}
		if cachedUrl != nil {
			ctx.JSON(http.StatusConflict, errResponse(errors.New("自定义短链接已被使用 (来自缓存)")))
			return
		}

		// 缓存未命中，检查数据库
		isAvailable, dbErr := server.store.IsShortCodeAvailable(ctx, shortCode)
		if dbErr != nil {
			log.Printf("ERROR: DB IsShortCodeAvailable for custom code '%s' failed: %v\n", shortCode, dbErr)
			ctx.JSON(http.StatusInternalServerError, errResponse(errors.New("服务内部错误")))
			return
		}
		if !isAvailable {
			ctx.JSON(http.StatusConflict, errResponse(errors.New("自定义短链接已被使用 (来自数据库)")))
			return
		}

	} else {
		// --- 分支二: 处理自动生成短链接 ---
		isCustom = false
		generatedCode, err := server.getUniqueShortCode(ctx)
		if err != nil {
			log.Printf("ERROR: Failed to generate unique short code: %v\n", err)
			ctx.JSON(http.StatusInternalServerError, errResponse(errors.New("生成短链接失败")))
			return
		}
		shortCode = generatedCode
	}

	// --- 统一的创建逻辑 ---
	// 此时 shortCode 无论是自定义的还是自动生成的，都已确保唯一
	urlParams := db.CreateUrlParams{
		ShortCode:   shortCode,
		IsCustom:    isCustom,
		OriginalUrl: req.OriginalURL,
		ExpiredAt:   finalExpireAt,
	}

	createdUrl, err := server.store.CreateUrl(ctx, urlParams)
	if err != nil {
		log.Printf("ERROR: DB CreateUrl for code '%s' failed: %v\n", shortCode, err)
		ctx.JSON(http.StatusInternalServerError, errResponse(errors.New("创建短链接失败")))
		return
	}

	// **【核心修改】**
	// 将新创建的链接信息写入缓存，统一调用 setUrlInCache 函数
	// 这个函数内部会使用 HSET
	go func() {
		// 使用后台协程，避免阻塞主请求
		err := server.setUrlInCache(ctx, &createdUrl)
		if err != nil {
			log.Printf("WARN: Failed to set cache for code '%s' after DB insert: %v\n", createdUrl.ShortCode, err)
		}
	}()

	// 成功响应
	ctx.JSON(http.StatusOK, model.CreateURLResponse{
		Success:   true,
		ShortCode: createdUrl.ShortCode,
		ExpireAt:  createdUrl.ExpiredAt,
	})
}
func (server *Server) RedirectURL(ctx *gin.Context) {
	shortcode := ctx.Query("shortcode")
	if shortcode == "" {
		ctx.JSON(http.StatusBadRequest, errResponse(errors.New("缺少 shortcode 参数")))
		return
	}
	// 先检查 Redis 缓存

	cachedUrl, err := server.getUrlFromCache(ctx, shortcode)
	// originalURL, err := server.rdb.Get(ctx, redisKey).Result()
	if err == nil { // 缓存命中！
		// 直接使用缓存数据进行重定向
		go server.recordClick(cachedUrl.ID) // 异步记录点击
		ctx.Redirect(http.StatusFound, cachedUrl.OriginalUrl)
		return
	}
	if err != redis.Nil { // 遇到了真正的 Redis 错误
		// 记录错误日志，并可以考虑重定向到一个错误页面
		log.Printf("Redis error: %v", err)
	}

	url, dbErr := server.getUrlByDB(ctx, shortcode)
	if dbErr == nil {
		// 检查是否过期
		if time.Now().After(url.ExpiredAt) {
			ctx.JSON(http.StatusGone, errResponse(errors.New("短链接已过期")))
			return
		}
		go server.recordClick(url.ID)
		ctx.Redirect(http.StatusFound, url.OriginalUrl)
		go server.setUrlInCache(ctx, &url)
		return

	} else if dbErr == util.ErrNotFoundInDB {
		ctx.JSON(http.StatusNotFound, dbErr.Error())
		return
	} else {
		ctx.JSON(http.StatusInternalServerError, dbErr.Error())
		return
	}

}
func (server *Server) recordClick(urlID int64) {
	select {
	case server.clickChan <- urlID:
	default:
		log.Println("WARN: Click channel is full. Discarding click event.")
	}
}
func (server *Server) ClickProcessor() {
	// 使用 map 在内存中聚合点击次数
	// key: url_id, value: click_count
	clicks := make(map[int64]int)
	// 创建一个定时器，例如每5秒触发一次，将聚合数据刷入数据库
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	log.Println("INFO: Starting background click processor...")

	for {
		select {
		case urlID := <-server.clickChan:
			// 每接收到一个点击，就在 map 中累加
			clicks[urlID]++
		case <-ticker.C:
			// 定时器触发
			if len(clicks) == 0 {
				continue
			}

			// 为了不阻塞后续的 channel 接收，
			// 将当前聚合的 map 复制出来，并重置原来的 map
			clicksToFlush := clicks
			clicks = make(map[int64]int)

			// 在一个新的 goroutine 中执行数据库写入操作，
			// 以免长时间的DB操作阻塞 clickProcessor
			go server.flushClicksToDB(clicksToFlush)
		}
	}
}
func (server *Server) flushClicksToDB(clicksToFlush map[int64]int) {
	err := server.store.AddUrlClicks(context.Background(), clicksToFlush)

	if err != nil {
		log.Printf("ERROR: Failed to flush clicks to DB: %v", err)

	}

}

func (s *Server) setUrlInCache(ctx *gin.Context, urlData *db.Url) error {
	// 检查传入的数据是否有效
	if urlData == nil || urlData.ShortCode == "" {
		return fmt.Errorf("invalid url data provided")
	}

	// 将整个 Url 结构体序列化为 JSON 字符串
	payloadBytes, err := json.Marshal(urlData)
	if err != nil {
		return fmt.Errorf("failed to marshal url data: %w", err)
	}

	err = s.rdb.HSet(ctx, shortcode_prefix, urlData.ShortCode, payloadBytes).Err()
	if err != nil {
		return fmt.Errorf("failed to execute HSet on redis: %w", err)
	}

	return nil
}

// getUrlFromCache 从 Redis Hash 中读取并返回一个有效的 Url 对象
func (s *Server) getUrlFromCache(ctx *gin.Context, shortCode string) (*db.Url, error) {
	// 使用 HGet 从 Redis 读取 JSON 字符串
	payloadBytes, err := s.rdb.HGet(ctx, shortcode_prefix, shortCode).Bytes()
	if err == redis.Nil {
		// 缓存未命中 (Key或Field不存在)，这是正常情况，直接返回
		return nil, redis.Nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to execute HGet from redis: %w", err)
	}

	// 将 JSON 反序列化回 Url 结构体
	var cachedUrl db.Url
	if err := json.Unmarshal(payloadBytes, &cachedUrl); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached url data: %w", err)
	}

	// **核心逻辑：检查是否过期**
	// 1. ExpiredAt 字段不是零值 (意味着设置了过期时间)
	// 2. 当前时间在 ExpiredAt 之后
	if !cachedUrl.ExpiredAt.IsZero() && time.Now().After(cachedUrl.ExpiredAt) {
		// 缓存已过期！
		// 异步地从哈希中删除这个过期的字段，避免占用空间
		go s.rdb.HDel(ctx, shortcode_prefix, shortCode)

		// 像缓存未命中一样处理，通知调用者去数据库查找
		return nil, redis.Nil
	}

	// 缓存命中且有效，返回完整的 Url 对象
	return &cachedUrl, nil
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
func (server *Server) getUrlByDB(ctx *gin.Context, shortcode string) (db.Url, error) {
	url, dbErr := server.store.GetUrlByShortCode(ctx, shortcode)
	if dbErr != nil {
		if dbErr == sql.ErrNoRows {
			return db.Url{}, util.ErrNotFoundInDB
		} else {
			return db.Url{}, util.ErrDatabase
		}

	}
	return url, dbErr
}

// errResponse 是你自定义的错误响应函数，这里只是一个占位符
func errResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}
