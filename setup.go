package main

import "github.com/go-redis/redis"

var client *redis.Client

func InitRedis() {
	// Redis

	client = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	_, err := client.Ping().Result()
	if err != nil {
		panic(err)
	}

}
