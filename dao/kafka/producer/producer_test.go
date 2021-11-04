package producer

import (
	"log"
	"metric-index/conf"
	"os"
	"path/filepath"
	"testing"
)

func TestKfkInit(t *testing.T) {
	home, _ := os.UserHomeDir()
	err := os.Chdir(filepath.Join(home, "go", "metric-index"))
	if err != nil {
		panic(err)
	}

	if err := conf.Init(); err != nil {
		log.Printf("init settings failed, err:%v\n", err)
		return
	}
	log.Println(conf.Conf.MetricStore.Producer.Hosts)
	log.Println(conf.Conf.MetricStore.Producer.Topic)

	err = Init()
	log.Println(err)
}
