package routes

import (
	"metric-index/conf"
	"metric-index/controllers/metrics"
	"metric-index/utils/logger"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Init 路由配置
func Init(app *gin.Engine) {
	app.Use(logger.GinLogger(), logger.GinRecovery(true))

	app.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	app.Any("/health", func(ctx *gin.Context) { //healthCheck
		ctx.String(http.StatusOK, "SUCCESS")
	})

	if conf.Conf.AppConfig.RoleType == conf.RoleTypeProducer {
		// 注册controller/metrics的route
		metrics.InitRoute(app)
	}

}
