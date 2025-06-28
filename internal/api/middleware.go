package api

import (
	"encoding/json"
	"errors"
	"fmt"
	db "github/heimaolst/urlshorter/db/sqlc"
	"github/heimaolst/urlshorter/internal/auth"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

func (server *Server) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errResponse(errors.New("请求未包含授权令牌")))
			return
		}

		// 通常令牌格式是 "Bearer <token>"，我们需要解析出 <token> 部分
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			// log.Println("Auth -> ", authHeader)
			c.AbortWithStatusJSON(http.StatusUnauthorized, errResponse(errors.New("授权令牌格式不正确")))
			return
		}

		tokenString := parts[1]

		// 解析并验证令牌
		claims := &auth.Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return auth.JWTSecret, nil // 返回签名密钥
		})

		if err != nil {
			if err == jwt.ErrSignatureInvalid {
				c.AbortWithStatusJSON(http.StatusUnauthorized, errResponse(errors.New("无效的签名")))
				return
			}
			if errors.Is(err, jwt.ErrTokenExpired) || errors.Is(err, jwt.ErrTokenNotValidYet) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, errResponse(errors.New("令牌已过期或未激活")))
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, errResponse(errors.New("无法处理的令牌")))
			return
		}

		if !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errResponse(errors.New("无效的令牌")))
			return
		}

		userID := claims.UserID
		redisKey := fmt.Sprintf("user:%d", userID)

		// 1. 优先从 Redis 缓存中获取用户信息
		userDataJSON, err := server.rdb.Get(c, redisKey).Result()
		if err == nil {
			// 缓存命中！
			var user db.User
			json.Unmarshal([]byte(userDataJSON), &user)
			// 将用户信息存入 Context，继续请求
			c.Set("user", user)
			c.Next()
			return
		}

		if err != redis.Nil {
			// Redis 发生其他错误
			log.Printf("Redis error while getting user: %v", err)
		}

		// 2. 缓存未命中 (err == redis.Nil)，从数据库中查找
		user, dbErr := server.store.GetUserByID(c, userID)
		if dbErr != nil {
			// 数据库中也找不到该用户（可能已被删除），认证失败
			c.AbortWithStatusJSON(http.StatusUnauthorized, errResponse(errors.New("用户不存在")))
			return
		}

		// 3. 将从数据库中查到的数据写回缓存
		// 注意：写入缓存前，清空密码哈希，避免敏感信息进入缓存
		user.PasswordHash = ""
		userDataBytes, _ := json.Marshal(user)
		// 设置一个合理的过期时间，例如 1 小时
		server.rdb.Set(c, redisKey, userDataBytes, time.Hour*1)

		// 将用户信息存入 Context，继续请求
		c.Set("user", user)
		c.Next()
	}
}
