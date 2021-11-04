package metrics

import (
	"metric-index/dao/gocache"
	"metric-index/dao/kafka/producer"
	"metric-index/dao/redis"

	"github.com/prometheus/prometheus/prompb"

	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
)

// Metric 写入kafka的结构体
type Metric struct {
	Labels    []*prompb.Label `json:"labels"`
	Content   string          `json:"content"`
	IsInCache bool            `json:"-"`
}

// Store 存储metric，metric=metricName+label
// 查询redis中是否有metric中的md5，如果没有则插入
// 需要在controller做wq处理，metric组装（调用WQMetricFilterAndAsm或者AsmMetric）
func Store(wq *prompb.WriteRequest) {
	metrics := Assembler(wq)
	metricStrings := make([]string, 0, len(metrics))
	json := jsoniter.ConfigCompatibleWithStandardLibrary
	for i := range metrics {
		_, found := gocache.Get(metrics[i].Content)
		metrics[i].IsInCache = found
		if !found {
			metricStrings = append(metricStrings, metrics[i].Content)
		}
	}

	result, err := redis.PipeExistsByGet(metricStrings)
	if err != nil {
		zap.L().Error("check metric key is exist from redis failed", zap.Error(err))
	}

	j := 0
	for i := 0; i < len(metrics) && j < len(result); i++ {
		if metrics[i].IsInCache {
			continue
		}
		if result[j] {
			gocache.SetDefault(metrics[i].Content, 1)
		} else {
			metric, err := json.Marshal(metrics[j])
			if err != nil {
				zap.L().Error("metric struct marshal failed", zap.Error(err))
				j++
				continue
			}

			if err := producer.PushMessageAsync(metrics[i].Content, string(metric)); err != nil {
				zap.L().Error("metric send to kafka failed", zap.Error(err))
				continue
			}
		}
		j++
	}

	return
}
