package victoriametrics

import (
	"fmt"
	"testing"
)

func TestSend(t *testing.T) {
	url := "http://10.0.52.166:4242/api/put"
	contentType := "application/json"
	data := `{"timeseries":[{"metric":"zangkuo123","value":45.34,"tags":{"t1":"v1","t2":"v2"},"timestamp":1625560733}]}`
	err := send(url, contentType, []byte(data))
	fmt.Println(err)
}
