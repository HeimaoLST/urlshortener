package api

import (
	db "github/heimaolst/urlshorter/db/sqlc"

	"github.com/gin-gonic/gin"
)

type Server struct {
	store  *db.Store
	
	router *gin.Engine
}

func NewServer(store *db.Store) *Server {
	server := &Server{store: store}
	router := gin.Default()

	router.POST("/api/create", server.CreateURL)
	server.router = router

	return server

}

func (server *Server) Start(address string) error {
	return server.router.Run(address)

}

func errResponse(error error) gin.H {
	return gin.H{"error": error.Error()}
}
