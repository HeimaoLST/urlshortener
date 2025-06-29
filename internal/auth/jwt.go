package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// 使用不同的密钥来签发 Access Token 和 Refresh Token
var AccessTokenSecret = []byte("VERRRYRRRRACCETOKEEN")
var RefreshTokenSecret = []byte("RRRRRRRRREFRESHHHHHH")

// AccessTokenClaims 定义了 Access Token 中存储的数据
type AccessTokenClaims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// RefreshTokenClaims 定义了 Refresh Token 中存储的数据
type RefreshTokenClaims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

// GenerateAccessToken 生成一个新的 Access Token
func GenerateAccessToken(userID int64, username string) (string, error) {

	expirationTime := time.Now().Add(30 * time.Minute)

	claims := &AccessTokenClaims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(AccessTokenSecret)
}

// GenerateRefreshToken 生成一个新的 Refresh Token
func GenerateRefreshToken(userID int64) (string, error) {

	expirationTime := time.Now().Add(7 * 24 * time.Hour)

	claims := &RefreshTokenClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(RefreshTokenSecret)
}

// ValidateRefreshToken 解析并验证 Refresh Token
func ValidateRefreshToken(tokenString string) (*RefreshTokenClaims, error) {
	claims := &RefreshTokenClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		// 确保签名算法是预期的
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return RefreshTokenSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid refresh token")
	}

	return claims, nil
}
