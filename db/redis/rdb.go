package redis

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

func NewRedisClient(opt *redis.Options) *redis.Client {
	rdb := redis.NewClient(opt)

	pong, err := rdb.Ping(ctx).Result()

	if err != nil {
		log.Fatal("cannot connect to redis: ", err)
	}

	log.Println("已连接Redis", pong)
	return rdb
}
