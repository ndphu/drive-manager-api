package service

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"os"
	"time"
)

type RedisService struct {
	rdb *redis.Client
}

var ctx = context.Background()

var redisService *RedisService

func GetRedisService() (*RedisService, error) {
	if redisService == nil {
		rdb := redis.NewClient(&redis.Options{
			Addr:     os.Getenv("REDIS_SERVER"),
			Password: os.Getenv("REDIS_AUTH"), // no password set
			DB:       0,  // use default DB
		})

		redisService = &RedisService{
			rdb: rdb,
		}
	}
	return redisService, nil
}

func (s *RedisService) Save(key, value string) error {
	err := s.rdb.Set(ctx, key, value, 30*time.Minute).Err()
	return err
}


func (s *RedisService) Get(key string) (string, error) {
	value, err := s.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		fmt.Println(key, "does not exist")
		return "", nil
	} else if err != nil {
		return "", err
	} else {
		return value, nil
	}
}
