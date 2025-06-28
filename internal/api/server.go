package api

import (
	db "github/heimaolst/urlshorter/db/sqlc"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type Server struct {
	store *db.Store

	router *gin.Engine

	rdb *redis.Client

	clickChan chan int64
}

func NewServer(store *db.Store, rdb *redis.Client, size int) *Server {
	server := &Server{store: store,
		rdb:       rdb,
		clickChan: make(chan int64, size)}
	router := gin.Default()
	router.Use(cors.Default())

	router.POST("/api/users/login", server.Login)
	router.POST("/api/users/register", server.RegisterUser)
	// router.GET("/:shortcode", server.RedirectURL)
	api := router.Group("api")
	{
		api.GET("/jump", server.RedirectURL)

	}

	authRoutes := router.Group("/api").Use(server.AuthMiddleware())
	{
		authRoutes.POST("/create", server.CreateURL)
	}

	server.router = router

	return server

}

func (server *Server) Start(address string) error {
	return server.router.Run(address)

}
