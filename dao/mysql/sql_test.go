package mysql

import (
	"fmt"
	"log"
	"metric-index/conf"
	"metric-index/dao/gocache"
	"metric-index/dao/redis"
	"testing"
)

func TestSQL(t *testing.T) {
	if err := conf.Init(); err != nil {
		fmt.Printf("init settings failed, err:%v\n", err)
		return
	}
	fmt.Println(conf.Conf.MySQLConfig)
	if err := Init(conf.Conf.MySQLConfig); err != nil {
		fmt.Printf("init mysql failed, err:%v\n", err)
		return
	}
	if err := db.Ping(); err != nil {
		fmt.Println(err)
	}
	sql := "SELECT `id` FROM `metric_label_name` WHERE `name`='device'"
	ID, err := GetID(sql)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("ID", ID)
}

func TestRDS(t *testing.T) {
	if err := conf.Init(); err != nil {
		fmt.Printf("init settings failed, err:%v\n", err)
		return
	}

	// 初始化redis
	if err := redis.Init(conf.Conf.RedisConfig); err != nil {
		fmt.Printf("init redis failed, err:%v\n", err)
		return
	}

	res, err := redis.Get("abc")
	if err != nil {
		log.Println("err:", err)
	}

	log.Println("res", res)

	// 初始化gocache
	gocache.Init()

	aaa, found := gocache.Get("bac")
	if !found {
		log.Println("not found")
	}

	gocache.SetDefault("abc", 1)
	aaa, found = gocache.Get("abc")
	if found {
		log.Println("found it:", aaa)
	}

}
