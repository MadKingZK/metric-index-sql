package utils

import (
	"fmt"
	"testing"

	"github.com/go-redis/redis"
)

func TestRedisPipe(t *testing.T) {

	client := redis.NewClient(&redis.Options{
		Addr:     "10.0.52.166:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
		Network:  "tcp",
		PoolSize: 50,
	})

	if _, err := client.Ping().Result(); err != nil {
		panic(err)
	}

	pipe := client.Pipeline()

	pipe.Get("aa")
	pipe.Get("bb")
	pipe.Get("aa")
	pipe.Get("bb")
	pipe.Get("aa")
	pipe.Get("bb")
	pipe.Get("aa")
	pipe.Get("bb")
	pipe.Get("aa")
	pipe.Get("bb")
	pipe.Get("aa")
	var res []bool
	cmders, _ := pipe.Exec()
	for i := range cmders {
		cmd := cmders[i].(*redis.StringCmd)
		_, err := cmd.Int()
		res = append(res, err == nil)
	}

	fmt.Println(res)
	defer pipe.Close()

	defer client.Close()
}
