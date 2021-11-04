package victoriametrics

import (
	"bytes"
	"metric-index/conf"
	"net/http"
	"time"

	"github.com/pkg/errors"

	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
	"go.uber.org/zap"
)

// Write 处理remote write数据
func Write(wq *prompb.WriteRequest) (err error) {
	data, err := wq.Marshal()
	if err != nil {
		zap.L().Error("marshal prompb.WirteRequest falil: ", zap.Error(err))
	}

	b := snappy.Encode(nil, data)
	err = send(conf.Conf.Remote.Write.URL, conf.Conf.Remote.Write.ContentType, b)

	return
}

func send(url, contentType string, data []byte) (err error) {
	client := &http.Client{Timeout: 5 * time.Second}
	req := bytes.NewReader(data)
	resp, err := client.Post(url, contentType, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		err = errors.Errorf("send metrics failed")
		zap.L().Error("send metrics failed", zap.Int("status", resp.StatusCode))
		return err
	}
	return
}

// ReqForward 直接转发接受的request
func ReqForward(data []byte) (err error) {
	client := &http.Client{Timeout: 5 * time.Second}
	req := bytes.NewReader(data)
	resp, err := client.Post(conf.Conf.Remote.Write.URL, conf.Conf.Remote.Write.ContentType, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		err = errors.Errorf("send metrics failed")
		zap.L().Error("send metrics failed", zap.Int("status", resp.StatusCode))
		return err
	}
	return
}
