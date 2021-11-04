package main

import (
	"context"
	"fmt"
	"metric-index/conf"
	"metric-index/dao/gocache"
	"metric-index/dao/kafka/producer"
	"metric-index/dao/mysql"
	"metric-index/dao/redis"
	"metric-index/routes"
	"metric-index/services/worker"
	"metric-index/utils/logger"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/pprof"

	"github.com/gin-gonic/gin"

	"go.uber.org/zap"
)

func main() {
	// 初始化配置
	if err := conf.Init(); err != nil {
		fmt.Printf("init settings failed, err:%v\n", err)
		return
	}

	// 初始化日志
	if err := logger.Init(conf.Conf.LogConfig); err != nil {
		fmt.Printf("init logger failed, err:%v\n", err)
		return
	}
	defer zap.L().Sync()

	// 根据RoleType初始化producer或者comsummer
	if conf.Conf.AppConfig.RoleType == conf.RoleTypeProducer {
		if err := producerInit(); err != nil {
			return
		}
		fmt.Printf("producer worker started ...")
	} else if conf.Conf.AppConfig.RoleType == conf.RoleTypeConsummer {
		if err := consummerInit(); err != nil {
			return
		}
		fmt.Printf("consummer worker started ...")
	} else {
		fmt.Printf("can not get role type, please config role_type")
		zap.L().Error("can not get role type, please config role_type")
	}

	// 初始化路由
	app := gin.New()
	routes.Init(app)
	pprof.Register(app)

	// 启动服务（优雅关机）
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", conf.Conf.AppConfig.Port),
		Handler: app,
	}

	go func() {
		// 开启一个goroutine启动服务
		if err := srv.ListenAndServe(); err == nil && err != http.ErrServerClosed {
			zap.L().Fatal("listen: ", zap.Error(err))
		}
	}()

	// 等待中断信号来优雅地关闭服务器，为关闭服务器操作设置一个5秒的超时
	// 创建一个接收信号的通道
	quit := make(chan os.Signal, 1)

	// signal.Notify把收到的 syscall.SIGINT或syscall.SIGTERM 信号转发给quit
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	zap.L().Info("Shutdown Server ...")
	// 创建一个5秒超时的context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	defer consummerClose()
	defer producerClose()
	// 5秒内优雅关闭服务（将未处理完的请求处理完再关闭服务），超过5秒就超时退出
	if err := srv.Shutdown(ctx); err != nil {
		zap.L().Fatal("Server Shutdown: ", zap.Error(err))
	}

	zap.L().Info("Server exiting")
}

func producerInit() (err error) {
	// 初始化kafka
	if err = producer.Init(); err != nil {
		fmt.Println("init kafka failed, err:%v\n", err)
		return
	}

	// 初始化redis
	if err = redis.Init(conf.Conf.RedisConfig); err != nil {
		fmt.Printf("init redis failed, err:%v\n", err)
		return
	}

	// 初始化gocache
	gocache.Init()
	//defer gocache.Save()

	return
}

func producerClose() {
	redis.Close()
	redis.CloseCommitter()
	producer.Close()
}

func consummerInit() (err error) {
	// 初始化mysql
	if err = mysql.Init(conf.Conf.MySQLConfig); err != nil {
		fmt.Printf("init mysql failed, err:%v\n", err)
		return
	}

	// 初始化redis
	if err = redis.Init(conf.Conf.RedisConfig); err != nil {
		fmt.Printf("init redis failed, err:%v\n", err)
		return
	}

	// 初始化gocache
	gocache.Init()
	//defer gocache.Save()
	go worker.Run()
	return
}

func consummerClose() {
	mysql.Close()
}
