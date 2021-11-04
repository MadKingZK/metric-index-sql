package victoriametrics

import (
	apimetrics "metric-index/api/metrics"
	"metric-index/conf"

	jsoniter "github.com/json-iterator/go"
)

// Send 发送metrics到远端服务
func Send(req []*apimetrics.TimeSeries) (err error) {
	json := jsoniter.ConfigCompatibleWithStandardLibrary
	data, err := json.Marshal(req)
	if err != nil {
		return
	}

	err = send(conf.Conf.Remote.Send.URL, conf.Conf.Remote.Send.ContentType, data)

	return
}
