// api/user.go (一个新文件，或添加到现有文件中)
package api

import (
	"database/sql"
	"errors"
	db "github/heimaolst/urlshorter/db/sqlc"
	"github/heimaolst/urlshorter/internal/auth"
	"github/heimaolst/urlshorter/internal/util"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

// LoginRequest 定义了登录请求的结构
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// RegisterUserRequest 定义了用户注册时需要的请求体
type RegisterUserRequest struct {
	Username string `json:"username" binding:"required,alphanum,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// UserResponse 定义了返回给客户端的用户信息，隐藏了敏感数据
type UserResponse struct {
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// newUserResponse 是一个辅助函数，用于将数据库模型转换为安全的响应模型
func newUserResponse(user db.User) UserResponse {
	return UserResponse{
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}
}



// Login 处理用户登录，现在返回两种令牌
func (server *Server) Login(ctx *gin.Context) {
	var req LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errResponse(err))
		return
	}

	user, err := server.store.GetUserByUsername(ctx, req.Username)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusUnauthorized, errResponse(errors.New("用户名或密码错误")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errResponse(err))
		return
	}

	err = util.CheckPassword(req.Password, user.PasswordHash)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errResponse(errors.New("用户名或密码错误")))
		return
	}

	// 1. 生成 Access Token
	accessToken, err := auth.GenerateAccessToken(user.ID, user.Username)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errResponse(errors.New("生成访问令牌失败")))
		return
	}

	// 2. 生成 Refresh Token
	refreshToken, err := auth.GenerateRefreshToken(user.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errResponse(errors.New("生成刷新令牌失败")))
		return
	}

	// 3. 返回两种令牌
	rsp := LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
	ctx.JSON(http.StatusOK, rsp)
}

// --- 新增刷新令牌的 Handler ---

// RefreshTokenRequest 定义了刷新令牌接口的请求体
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RefreshToken 用于使用有效的 Refresh Token 获取新的 Access Token
func (server *Server) RefreshToken(ctx *gin.Context) {
	var req RefreshTokenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errResponse(err))
		return
	}

	// 1. 验证 Refresh Token
	claims, err := auth.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errResponse(errors.New("无效的刷新令牌")))
		return
	}

	// 2. 从数据库中获取最新的用户信息（确保用户没有被禁用或删除）
	user, err := server.store.GetUserByID(ctx, claims.UserID)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errResponse(errors.New("用户不存在")))
		return
	}

	// 3. 生成一个新的 Access Token
	newAccessToken, err := auth.GenerateAccessToken(user.ID, user.Username)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errResponse(errors.New("生成访问令牌失败")))
		return
	}

	// 4. 返回新的 Access Token
	ctx.JSON(http.StatusOK, gin.H{"access_token": newAccessToken})
}
func (server *Server) RegisterUser(ctx *gin.Context) {
	var req RegisterUserRequest
	// 1. 绑定并验证请求体
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errResponse(err))
		return
	}

	// 2. 对明文密码进行哈希处理
	hashedPassword, err := util.Crypto(req.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errResponse(errors.New("处理密码时出错")))
		return
	}

	// 3. 准备数据库插入参数
	params := db.CreateUserParams{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
	}

	// 4. 调用数据库层创建用户
	user, err := server.store.CreateUser(ctx, params)
	if err != nil {
		// 检查是否为 PostgreSQL 的唯一性冲突错误
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code.Name() == "unique_violation" {
				// 通过错误消息判断是哪个字段冲突了
				if strings.Contains(pqErr.Message, "users_username_key") {
					ctx.JSON(http.StatusConflict, errResponse(errors.New("该用户名已被使用")))
					return
				}
				if strings.Contains(pqErr.Message, "users_email_key") {
					ctx.JSON(http.StatusConflict, errResponse(errors.New("该邮箱已被注册")))
					return
				}
			}
		}
		// 其他数据库错误
		ctx.JSON(http.StatusInternalServerError, errResponse(errors.New("创建用户失败")))
		return
	}

	// 5. 成功创建，返回格式化后的用户信息 (不含密码)

	rsp := newUserResponse(user)
	ctx.JSON(http.StatusCreated, rsp)
}
