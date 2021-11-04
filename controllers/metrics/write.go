package metrics

import (
	"io/ioutil"
	"metric-index/dao/gocache"
	"metric-index/services/metrics"
	"net/http"

	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
	"github.com/golang/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
)

// Write 接收remote write
func Write(c *gin.Context) {
	cmpBody, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		zap.L().Error("read request.body failed", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": http.StatusText(http.StatusBadRequest),
			"data":    "",
		})
		return
	}
	//c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(cmpBody))

	// 解压cmp_body
	body, err := snappy.Decode(nil, cmpBody)
	if err != nil {
		zap.L().Error("uncompress request.body failed", zap.Error(err))
		c.JSON(http.StatusTooManyRequests, gin.H{
			"code":    http.StatusTooManyRequests,
			"message": http.StatusText(http.StatusTooManyRequests),
			"data":    "",
		})
		return
	}

	var wq = new(prompb.WriteRequest)
	if err = proto.Unmarshal(body, wq); err != nil {
		panic(err)
	}

	metrics.Store(wq)

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "success",
		"data":    "",
	})
	return
}

// CacheResp ...
type CacheResp struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
	Count int         `json:"count"`
}

// Cache get cache by key
func Cache(c *gin.Context) {
	cache := new(CacheResp)
	cache.Key = c.Query("key")
	var found bool
	cache.Value, found = gocache.Get(cache.Key)
	cache.Count = gocache.Count()
	if !found {
		c.JSON(http.StatusOK, gin.H{
			"code":    http.StatusOK,
			"message": http.StatusText(http.StatusNotFound),
			"data":    cache,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": http.StatusText(http.StatusOK),
		"data":    cache,
	})
}
