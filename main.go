package main

import (
	"context"
	"database/sql"
	redisclient "github/heimaolst/urlshorter/db/redis"
	db "github/heimaolst/urlshorter/db/sqlc"
	"github/heimaolst/urlshorter/internal/api"
	"github/heimaolst/urlshorter/internal/util"
	"log"
	
	_ "github.com/lib/pq"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

func main() {
	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal("cannot load config: ", err)
	}

	conn, err := sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		log.Fatal("cannot connect to db: ", err)
	}
	log.Println("已连接数据库")
	opt, err := redis.ParseURL(config.RedisAddress)
	if err != nil {
		panic(err)
	}

	store := db.NewStore(conn)

	rdb := redisclient.NewRedisClient(opt)

	server := api.NewServer(store, rdb)

	server.Start(config.ServerAddress)
}
