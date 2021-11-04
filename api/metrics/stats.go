package metrics

// StatsResp 获取es bulk api状态
type StatsResp struct {
	NumAdded    uint64 `json:"num_added"`
	NumFlushed  uint64 `json:"num_flushed"`
	NumFailed   uint64 `json:"num_failed"`
	NumIndexed  uint64 `json:"num_indexed"`
	NumCreated  uint64 `json:"num_created"`
	NumUpdated  uint64 `json:"num_updated"`
	NumDeleted  uint64 `json:"num_deleted"`
	NumRequests uint64 `json:"num_requests"`
}
