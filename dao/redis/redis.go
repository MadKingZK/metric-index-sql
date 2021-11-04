package redis

import (
	"context"
	"errors"
	"fmt"
	"metric-index/conf"
	"time"

	"go.uber.org/zap"

	"github.com/go-redis/redis"
)

// 声明一个全局的rdb变量
var rdb *redis.Client
var ci Committer

// Init 初始化redis
func Init(cfg *conf.RedisConfig) (err error) {
	rdb = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%d",
			cfg.Host,
			cfg.Port,
		),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	if _, err = rdb.Ping().Result(); err != nil {
		return
	}

	err = InitCommitter()
	return
}

// InitCommitter 初始化redis committer
func InitCommitter() (err error) {
	cfg := CommitterConfig{
		NumWorkers:    conf.Conf.MetricStore.Cache.WorkerNum,
		FlushLens:     conf.Conf.MetricStore.Cache.FlushLens,
		FlushInterval: conf.Conf.MetricStore.Cache.FlushInterval * time.Second,
		Client:        rdb,
		DebugLogger:   nil,
	}
	cfg.ActionSet()
	ci, err = NewCommitter(cfg)
	if err != nil {
		zap.L().Error("Error creating the indexer: %s", zap.Error(err))
	}
	return
}

// Close 关闭redis连接
func Close() {
	_ = rdb.Close()
}

// CloseCommitter 关闭committer，在main中defer调用
func CloseCommitter() {
	if err := ci.Close(context.Background()); err != nil {
		zap.L().Error("Close Committer Failed", zap.Error(err))
	}
	return
}

// Push 推送redis cmd
func Push(commitItem CommitItem) (err error) {
	err = ci.Push(context.Background(), commitItem)
	return
}

// ExistOne 判断一个key是否在redis中存在
func ExistOne(key string) (res bool, err error) {
	cnt, err := rdb.Exists(key).Result()
	if err != nil {
		return false, err
	}
	if cnt != 1 {
		return false, nil
	}
	return true, nil
}

// Get 获取单个key的value
func Get(key string) (value string, err error) {
	value, err = rdb.Get(key).Result()
	return
}

// PipeExistsByGet 通过get判断简单kv类型的key是否存在
func PipeExistsByGet(keys []string) ([]bool, error) {
	if len(keys) <= 0 {
		return nil, nil
	}
	pipe := rdb.Pipeline()
	for i := range keys {
		pipe.Get(keys[i])
	}

	result := make([]bool, len(keys))
	cmders, _ := pipe.Exec()

	for i := range cmders {
		cmd, ok := cmders[i].(*redis.StringCmd)
		if !ok {
			return nil, errors.New("interface conversion: cat not convert redis.StringCmd")
		}
		_, err := cmd.Int()
		result[i] = err == nil
	}
	defer pipe.Close()
	return result, nil
}

// PipeSetNX 通过get判断简单kv类型的key是否存在
func PipeSetNX(keys []string, v interface{}, exTime time.Duration) ([]bool, error) {
	if len(keys) <= 0 {
		return nil, nil
	}
	pipe := rdb.Pipeline()
	for i := range keys {
		pipe.SetNX(keys[i], v, exTime)
	}

	result := make([]bool, len(keys))
	cmders, _ := pipe.Exec()

	for i := range cmders {
		cmd, ok := cmders[i].(*redis.BoolCmd)
		if !ok {
			return nil, errors.New("interface conversion: cat not convert redis.StringCmd")
		}
		res, _ := cmd.Result()
		result[i] = res
	}
	defer pipe.Close()
	return result, nil
}

// Set 插入string类型数据
func Set(k string, v interface{}, exTime time.Duration) (err error) {
	_, err = rdb.Set(k, v, exTime).Result()
	return
}

// PipeSet 批量set value
func PipeSet(kvs map[string]interface{}, exTime time.Duration) (err error) {
	pipe := rdb.Pipeline()
	for k, v := range kvs {
		pipe.Set(k, v, exTime)
	}

	_, err = pipe.Exec()
	defer pipe.Close()
	return

}

// SetNX 在指定的key不存在时，为key设置指定的值
// key不存在，设置成功，返回1；key存在，设置失败，返回0
func SetNX(key string, value interface{}, exTime time.Duration) (isSet bool, err error) {
	isSet, err = rdb.SetNX(key, value, exTime).Result()
	return
}

// Del 删除redis的key
func Del(key string) (err error) {
	_, err = rdb.Del(key).Result()
	return
}
