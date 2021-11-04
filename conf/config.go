package conf

import (
	"fmt"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Conf 保存配置的全局变量
var Conf = new(Config)

// Config 配置入口
type Config struct {
	*AppConfig   `mapstructure:"app"`
	*LogConfig   `mapstructure:"log"`
	*MySQLConfig `mapstructure:"mysql"`
	*RedisConfig `mapstructure:"redis"`
	*Remote      `mapstructure:"remote"`
	*MetricStore `mapstructure:"metric_store"`
}

// AppConfig 项目配置
type AppConfig struct {
	Name     string `mapstructure:"name"`
	Mode     string
	RoleType RoleType `mapstructure:"role_type"`
	Version  string   `mapstructure:"version"`
	Port     int      `mapstructure:"port"`
}

//RoleType 当前节点角色
type RoleType string

const (
	// RoleTypeProducer 以producer模式启动
	RoleTypeProducer RoleType = "producer"
	// RoleTypeConsummer 以consummer模式启动
	RoleTypeConsummer = "consummer"
)

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level"`
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxAge     int    `mapstructure:"max_age"`
	MaxBackups int    `mapstructure:"max_backups"`
}

// MySQLConfig mysql数据库配置
type MySQLConfig struct {
	Host         string `mapstructure:"host"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	DBName       string `mapstructure:"db_name"`
	Port         int    `mapstructure:"port"`
	MaOpenConns  int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

// RedisConfig redis配置
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

// MetricStore 监控指标名存储配置
type MetricStore struct {
	*Cache     `mapstructure:"cache"`
	*Producer  `mapstructure:"producer"`
	*Consummer `mapstructure:"consummer"`
}

// Cache 缓存配置
type Cache struct {
	IsExpire        bool          `mapstructure:"isexpire"`
	Expire          int           `mapstructure:"expire"`
	DistInterval    int           `mapstructure:"dist_interval"`
	DefaultExpire   time.Duration `mapstructure:"default_expire"`
	CleanupInterval time.Duration `mapstructure:"cleanup_interval"`
	WorkerNum       int           `mapstructure:"worker_num"`
	FlushLens       int           `mapstructure:"flush_lens"`
	FlushInterval   time.Duration `mapstructure:"flush_interval"`
}

// Producer kafka Producer配置
type Producer struct {
	Hosts []string `mapstructure:"hosts"`
	Topic string   `mapstructure:"topic"`
}

// OffsetType Consummer消费topic方式，起始/最新位置读取
type OffsetType string

const (
	// OffsetNewest 从最新位置开始读取
	OffsetNewest OffsetType = "newest"
	// OffsetOldest 从起始位置开始读取
	OffsetOldest = "oldest"
)

// Consummer kafka Consummer配置
type Consummer struct {
	Hosts      []string   `mapstructure:"hosts"`
	Topics     []string   `mapstructure:"topics"`
	GroupID    string     `mapstructure:"group_id"`
	WorkNum    int        `mapstructure:"work_num"`
	OffsetType OffsetType `mapstructure:"offset_type"`
}

// Remote 转发metrics到远端服务的相关配置
type Remote struct {
	*Write `mapstructure:"write"`
	*Send  `mapstructure:"send"`
}

// Write VictoriaMetrics remote write配置
type Write struct {
	URL         string `mapstructure:"url"`
	ContentType string `mapstructure:"content_type"`
}

// Send VictoriaMetrics /api/put 发送数据配置
type Send struct {
	URL         string `mapstructure:"url"`
	ContentType string `mapstructure:"content_type"`
}

// Init 初始化配置
func Init() (err error) {
	//viper.SetConfigFile("config.yaml")
	env := os.Getenv("GO_ENV")
	viper.SetConfigName(env)
	viper.SetConfigType("yml")
	viper.AddConfigPath("./conf/")
	if err = viper.ReadInConfig(); err != nil {
		fmt.Println("viper.ReadInConfig() ")
	}

	// 反序列化配置到全局变量Conf中
	if err := viper.Unmarshal(Conf); err != nil {
		fmt.Printf("viper.Unmarshal failed, err: %v\n", err)
	}

	Conf.Mode = env

	viper.WatchConfig()
	viper.OnConfigChange(func(in fsnotify.Event) {
		fmt.Println("config has already update...")
		// 反序列化配置更新到全局变量Conf中
		if err := viper.Unmarshal(Conf); err != nil {
			fmt.Printf("viper.Unmarshal failed, err: %v\n", err)
		}
	})

	return
}
