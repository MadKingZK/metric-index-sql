package worker

import (
	"metric-index/conf"
	"metric-index/dao/kafka/consummer"
	"metric-index/services/metrics"

	jsoniter "github.com/json-iterator/go"

	"go.uber.org/zap"
)

type handler struct {
}

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func (h handler) WorkHandler(msg []byte) {
	metric := &metrics.Metric{}
	err := json.Unmarshal(msg, metric)
	if err != nil {
		zap.L().Debug("WorkPoolRun err: ", zap.Error(err), zap.String("msg", string(msg)))
		return
	}
	zap.L().Debug("read kafka msg", zap.String("msg", string(msg)))
	Store(metric)
}

// Run 从kafka中消费数据
func Run() {
	consumer, err := consummer.NewConsumer(
		conf.Conf.MetricStore.Consummer.Hosts,
		conf.Conf.MetricStore.Consummer.GroupID,
		conf.Conf.MetricStore.Consummer.Topics,
		conf.Conf.MetricStore.Consummer.WorkNum)
	if err != nil {
		zap.L().Error("kafka Consummer Worker init err: ", zap.Error(err))
		return
	}
	consumer.Register(handler{})
	consumer.Run()
	return
}
