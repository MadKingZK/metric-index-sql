package metrics

// WriteReq 主动发送metric信息的接口请求
type WriteReq struct {
	Timeseries []*TimeSeries `json:"timeseries" binding:"required,dive,required"`
}

// TimeSeries 时序数据
type TimeSeries struct {
	MetricName string            `json:"metric" binding:"required"`
	Timestamp  int64             `json:"timestamp" binding:"omitempty"`
	Value      interface{}       `json:"value" binding:"required"`
	Labels     map[string]string `json:"tags" binding:"omitempty"`
}
