package gocache

import (
	"metric-index/conf"
	"time"

	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
)

var goCache *cache.Cache

// Init 初始化gocache
func Init() {
	goCache = cache.New(conf.Conf.MetricStore.Cache.DefaultExpire*time.Second, conf.Conf.MetricStore.Cache.CleanupInterval*time.Second)
	//if err := goCache.LoadFile("/opt/metric-index/cache.bat"); err != nil {
	//	zap.L().Error("load cache from file err", zap.Error(err))
	//}
	//go SaveByTimer()
}

// Set 设置k，v
func Set(k string, v interface{}, d time.Duration) {
	goCache.Set(k, v, d)
}

// SetDefault 设置k, v 使用默认过期时间
func SetDefault(k string, v interface{}) {
	goCache.SetDefault(k, v)
}

// Get get value
func Get(k string) (interface{}, bool) {
	v, found := goCache.Get(k)
	return v, found
}

// Count 统计key个数
func Count() (cnt int) {
	cnt = goCache.ItemCount()
	return
}

// Save 保存缓存到文件
func Save() error {
	err := goCache.SaveFile("/opt/metric-index/cache.bat")
	return err
}

// SaveByTimer 定时保存缓存
func SaveByTimer() {
	timer := time.NewTimer(5 * time.Minute)

	for {
		select {
		case <-timer.C:
			zap.L().Info("save gocache to file cache.bat")
			if err := Save(); err != nil {
				zap.L().Error("save gocache err", zap.Error(err))
			}
			timer.Reset(5 * time.Second)
		}
	}
}
