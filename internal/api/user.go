// api/user.go (一个新文件，或添加到现有文件中)
package api

import (
	"errors"
	db "github/heimaolst/urlshorter/db/sqlc"
	"github/heimaolst/urlshorter/internal/auth"
	"github/heimaolst/urlshorter/internal/util"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// LoginRequest 定义了登录请求的结构
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Email    string `json:"email" binding:"required"`
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

func (server *Server) Login(ctx *gin.Context) {
	var req LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errResponse(err))
		return
	}

	// 1. 从数据库根据 username 查询用户
	user, err := server.store.GetUserByUsername(ctx, req.Username)
	if err != nil {
		// ...处理用户不存在或数据库错误...
		ctx.JSON(http.StatusUnauthorized, errResponse(errors.New("用户名或密码错误")))
		return
	}

	// 2. 验证密码 (你需要一个密码验证的工具函数)
	err = util.CheckPassword(req.Password, user.PasswordHash)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errResponse(err))
		return
	}

	// 3. 密码验证成功，生成 JWT
	token, err := auth.GenerateToken(user.ID, user.Username)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errResponse(errors.New("生成令牌失败")))
		return
	}

	// 4. 返回令牌
	ctx.JSON(http.StatusOK, gin.H{"token": token})
}
func (server *Server) RegisterUser(ctx *gin.Context) {
	var req RegisterUserRequest
	// 1. 绑定并验证请求体
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errResponse(err))
		return
	}

	// 2. 对明文密码进行哈希处理
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
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
