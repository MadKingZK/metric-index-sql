package metrics

import (
	"bytes"

	"github.com/prometheus/prometheus/prompb"
)

// Assembler 组装metric
func Assembler(wq *prompb.WriteRequest) []Metric {
	if len(wq.Timeseries) == 0 {
		return nil
	}
	metrics := make([]Metric, len(wq.Timeseries))
	var b bytes.Buffer
	var flag bool
	for i := range wq.Timeseries {
		metrics[i].Labels = wq.Timeseries[i].Labels
		flag = false
		b.Reset()
		b.WriteString(wq.Timeseries[i].Labels[0].Value)
		b.WriteString(`{`)

		for j, l := 1, len(wq.Timeseries[i].Labels); j < l; j++ {
			b.WriteString(wq.Timeseries[i].Labels[j].Name)
			b.WriteString(`="`)
			b.WriteString(wq.Timeseries[i].Labels[j].Value)
			b.WriteString(`",`)
			flag = true

		}

		if flag {
			bt := b.Bytes()
			bt[len(bt)-1] = []byte("}")[0]
			metrics[i].Content = string(bt)
		} else {
			b.WriteString(`}`)
			metrics[i].Content = b.String()
		}
	}
	return metrics
}

//func bytes2str(b []byte) string {
//	return *(*string)(unsafe.Pointer(&b))
//}
