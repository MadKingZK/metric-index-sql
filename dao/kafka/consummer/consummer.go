package consummer

import (
	"errors"
	cfg "metric-index/conf"

	"github.com/Shopify/sarama"
	cluster "github.com/bsm/sarama-cluster"
	"go.uber.org/zap"
)

// Consumer 消费者
type Consumer struct {
	hosts        []string
	groupID      string
	topics       []string
	WorkNum      int
	handler      Handler
	Client       *cluster.Consumer
	saramaClient *cluster.Client
}

// NewConsumer 初始化消费者
func NewConsumer(hosts []string, groupID string, topics []string, workNum int) (*Consumer, error) {
	config := cluster.NewConfig()
	config.Group.Mode = cluster.ConsumerModePartitions
	if cfg.Conf.MetricStore.Consummer.OffsetType == cfg.OffsetNewest {
		config.Consumer.Offsets.Initial = sarama.OffsetNewest
	} else if cfg.Conf.MetricStore.Consummer.OffsetType == cfg.OffsetOldest {
		config.Consumer.Offsets.Initial = sarama.OffsetOldest
	} else {
		err := errors.New("err consummer offsets type set")
		zap.L().Error("", zap.Error(err))
		return nil, err
	}
	config.Consumer.Return.Errors = true
	config.Group.Return.Notifications = true
	saramaClient, err := cluster.NewClient(hosts, config)
	if err != nil {
		return nil, err
	}
	client, err := cluster.NewConsumerFromClient(saramaClient, groupID, topics)
	if err != nil {
		return nil, err
	}

	return &Consumer{
		hosts:        hosts,
		groupID:      groupID,
		topics:       topics,
		Client:       client,
		WorkNum:      workNum,
		saramaClient: saramaClient,
	}, nil
}

// Close 关闭消费者
func (consumer *Consumer) Close() {
	if consumer.Client == nil {
		return
	}
	err := consumer.Client.Close()
	if err != nil {
		return
	}
}

// Register 注册handler
func (consumer *Consumer) Register(h Handler) {
	consumer.handler = h
}

// Run 开始消费
func (consumer *Consumer) Run() {
	if consumer.Client == nil {
		return
	}
	defer consumer.Close()

	for {
		select {
		case part, ok := <-consumer.Client.Partitions():
			if !ok {
				return
			}

			zap.L().Info("beigin handle partition,", zap.Int32("Partition", part.Partition()))
			for workNum := 0; workNum < consumer.WorkNum; workNum++ {
				go consumer.readFromPart(part)
			}
		case err := <-consumer.Client.Errors():
			zap.L().Error("consumer err", zap.Error(err))
		case not := <-consumer.Client.Notifications():
			zap.L().Warn("consumer Notifications", zap.Any("not", not))
		}
	}
}

func (consumer *Consumer) readFromPart(pc cluster.PartitionConsumer) {
	for {
		select {
		case msg, ok := <-pc.Messages():
			if !ok {
				return
			}
			consumer.handler.WorkHandler(msg.Value)
			consumer.Client.MarkOffset(msg, "")
		}
	}
}
