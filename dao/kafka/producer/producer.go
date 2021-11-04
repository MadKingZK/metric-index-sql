package producer

import (
	"math/rand"
	"metric-index/conf"
	"metric-index/dao/gocache"
	"metric-index/dao/redis"
	"time"

	"go.uber.org/zap"

	"github.com/Shopify/sarama"
)

var syncProducerClient sarama.SyncProducer
var asyncProducerClient sarama.AsyncProducer

func newSyncProducer(hosts []string) (sarama.SyncProducer, error) {
	cfg := sarama.NewConfig()
	cfg.Producer.Retry.Max = 0
	cfg.Producer.RequiredAcks = sarama.WaitForAll
	cfg.Producer.Partitioner = sarama.NewRandomPartitioner
	cfg.Producer.Return.Successes = true
	client, err := sarama.NewClient(hosts, cfg)
	asyncClient, err := sarama.NewSyncProducerFromClient(client)
	if err != nil {
		return nil, err
	}
	return asyncClient, nil
}

// PushMessageSync 推送数据到kafka
func PushMessageSync(key, value string) (err error) {
	if syncProducerClient == nil {
		panic("producer run *p is nil")
	}
	sendMsg := &sarama.ProducerMessage{}
	sendMsg.Topic = conf.Conf.Producer.Topic
	sendMsg.Key = sarama.StringEncoder(key)
	sendMsg.Value = sarama.StringEncoder(value)
	_, _, err = syncProducerClient.SendMessage(sendMsg)
	return
}

func newAsyncProducer(hosts []string) (sarama.AsyncProducer, error) {
	cfg := sarama.NewConfig()
	cfg.Producer.Retry.Max = 0
	cfg.Producer.RequiredAcks = sarama.WaitForAll
	cfg.Producer.Partitioner = sarama.NewRandomPartitioner
	cfg.Producer.Return.Successes = true
	client, err := sarama.NewClient(hosts, cfg)
	asyncClient, err := sarama.NewAsyncProducerFromClient(client)
	if err != nil {
		return nil, err
	}
	return asyncClient, nil
}

func successHandler() {
	for {
		select {
		case success := <-asyncProducerClient.Successes():
			key, err := success.Key.Encode()
			if err != nil {
				zap.L().Error("cat not assert metadata to string", zap.Error(err))
			}
			bulkOnSuccess(string(key))
		}
	}
}

// PushMessageAsync 推送数据到kafka
func PushMessageAsync(key, value string) (err error) {
	if asyncProducerClient == nil {
		panic("producer run *p is nil")
	}
	sendMsg := &sarama.ProducerMessage{}
	sendMsg.Topic = conf.Conf.Producer.Topic
	sendMsg.Key = sarama.StringEncoder(key)
	sendMsg.Value = sarama.StringEncoder(value)
	asyncProducerClient.Input() <- sendMsg
	return
}

// PushMessagesSync 推送数据到kafka
func PushMessagesSync(values []string) (err error) {
	if syncProducerClient == nil {
		panic("producer run *p is nil")
	}
	msgs := make([]*sarama.ProducerMessage, len(values))
	for i := range values {
		sendMsg := &sarama.ProducerMessage{}
		sendMsg.Topic = conf.Conf.Producer.Topic
		sendMsg.Value = sarama.StringEncoder(values[i])
		msgs[i] = sendMsg
	}
	err = syncProducerClient.SendMessages(msgs)
	return
}

// Init 初始化kafka
func Init() (err error) {
	syncProducerClient, err = newSyncProducer(conf.Conf.Producer.Hosts)
	asyncProducerClient, err = newAsyncProducer(conf.Conf.Producer.Hosts)
	go successHandler()
	return
}

// Close 关闭kafka连接
func Close() {
	syncProducerClient.Close()
	asyncProducerClient.Close()
}

// bulkOnSuccess bulk成功时回调
func bulkOnSuccess(medata string) {
	// 配合PipeExistsByGet打开
	var exTime time.Duration
	if conf.Conf.MetricStore.Cache.IsExpire {
		exTime = time.Duration(conf.Conf.MetricStore.Cache.Expire-
			conf.Conf.MetricStore.Cache.DistInterval+
			rand.Intn(conf.Conf.MetricStore.Cache.DistInterval)) *
			time.Second
	} else {
		exTime = time.Duration(-1) * time.Second
	}

	gocache.SetDefault(medata, 1)
	if err := redis.Push(redis.CommitItem{
		Key:    medata,
		Value:  1,
		ExTime: exTime,
	}); err != nil {
		zap.L().Error("push metric to redis committer failed", zap.Error(err))
	}
}

// bulkOnFailure bulk失败时回调
func bulkOnFailure(medata string, err error) {
	// 插入ES失败，则删除redis记录
	zap.L().Error("insert into elasticsearch failed", zap.Error(err))
}
