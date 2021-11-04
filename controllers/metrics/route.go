package metrics

import "github.com/gin-gonic/gin"

// InitRoute 路由配置
func InitRoute(app *gin.Engine) {
	group := app.Group("/api/v1/metrics")
	group.POST("/write", Write)
	group.GET("/cache", Cache)
}
