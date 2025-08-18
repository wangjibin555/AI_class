package redis

import (
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

func InitRedis() (*redis.Client, error) {
	host := viper.GetString("redis.host")
	port := viper.GetInt("redis.port")
	password := viper.GetString("redis.password")
	db := viper.GetInt("redis.db")
	
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", host, port),
		Password: password,
		DB:       db,
	})
	
	return rdb, nil
}