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
}

func NewServer(store *db.Store, rdb *redis.Client) *Server {
	server := &Server{store: store,
		rdb: rdb}
	router := gin.Default()
	router.Use(cors.Default())
	api := router.Group("api")
	{
		api.POST("/create", server.CreateURL)

		api.GET("/jump", server.RedirectURL)

	}

	server.router = router

	return server

}

func (server *Server) Start(address string) error {
	return server.router.Run(address)

}
