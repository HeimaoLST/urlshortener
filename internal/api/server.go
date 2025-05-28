package api

import (
	db "github/heimaolst/urlshorter/db/sqlc"

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

	router.POST("/api/create", server.CreateURL)
	router.GET("/api/jump", server.RedirectURL)
	server.router = router

	return server

}

func (server *Server) Start(address string) error {
	return server.router.Run(address)

}

