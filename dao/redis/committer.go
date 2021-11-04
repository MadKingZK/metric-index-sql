package redis

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis"
)

// Committer redis批量提交
type Committer interface {
	Push(context.Context, CommitItem) error
	Close(context.Context) error
	Stats() CommitterStats
}

// CommitterConfig Committer配置
type CommitterConfig struct {
	NumWorkers    int           // The number of workers. Defaults to runtime.NumCPU().
	FlushLens     int           // The flush threshold in lens. Defaults to 1k.
	FlushInterval time.Duration // The flush threshold as duration. Defaults to 10sec.
	Client        *redis.Client // The Redis client.
	action        string
	DebugLogger   BulkIndexerDebugLogger // An optional logger for debugging.
}

// ActionSet 设置action
func (ccf *CommitterConfig) ActionSet() {
	ccf.action = "set"
}

// ActionGet 设置action
func (ccf *CommitterConfig) ActionGet() {
	ccf.action = "get"
}

// CommitterStats 记录Committer的状态
type CommitterStats struct {
	NumAdded    uint64
	NumDeleted  uint64
	NumFlushed  uint64
	NumFailed   uint64
	NumRequests uint64
}

// CommitItem 每个指令的结构
type CommitItem struct {
	Key    string
	Value  interface{}
	ExTime time.Duration
}

// BulkIndexerDebugLogger debug日志
type BulkIndexerDebugLogger interface {
	Printf(string, ...interface{})
}

type committer struct {
	wg      sync.WaitGroup
	queue   chan CommitItem
	workers []*worker
	ticker  *time.Ticker
	done    chan bool
	stats   *committerStats
	config  CommitterConfig
}

type committerStats struct {
	numAdded    uint64
	numDeleted  uint64
	numFlushed  uint64
	numFailed   uint64
	numRequests uint64
}

// NewCommitter 创建新committer
func NewCommitter(cfg CommitterConfig) (Committer, error) {
	if cfg.Client == nil {
		return nil, errors.New("redis client is nil")
	}

	if cfg.action == "" {
		return nil, errors.New("you need config action")
	}

	if cfg.NumWorkers == 0 {
		cfg.NumWorkers = runtime.NumCPU()
	}

	if cfg.FlushLens == 0 {
		cfg.FlushLens = 1000
	}

	if cfg.FlushInterval == 0 {
		cfg.FlushInterval = 10 * time.Second
	}

	ci := committer{
		config: cfg,
		done:   make(chan bool),
		stats:  &committerStats{},
	}

	ci.init()

	return &ci, nil
}

// Add 推送新item
func (ci *committer) Push(ctx context.Context, item CommitItem) error {
	atomic.AddUint64(&ci.stats.numAdded, 1)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case ci.queue <- item:
	}

	return nil
}

// Close 关闭committer
func (ci *committer) Close(ctx context.Context) error {
	ci.ticker.Stop()
	close(ci.queue)
	ci.done <- true

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		ci.wg.Wait()
	}

	for _, w := range ci.workers {
		w.mu.Lock()
		if len(w.items) > 0 {
			if err := w.flush(ctx); err != nil {
				w.mu.Unlock()
				continue
			}
		}
		w.mu.Unlock()
	}
	return nil
}

// Stats 获取committer状态
func (ci *committer) Stats() CommitterStats {
	return CommitterStats{
		NumAdded:    atomic.LoadUint64(&ci.stats.numAdded),
		NumDeleted:  atomic.LoadUint64(&ci.stats.numDeleted),
		NumFlushed:  atomic.LoadUint64(&ci.stats.numFlushed),
		NumFailed:   atomic.LoadUint64(&ci.stats.numFailed),
		NumRequests: atomic.LoadUint64(&ci.stats.numRequests),
	}
}

// init initializes the bulk indexer.
func (ci *committer) init() {
	ci.queue = make(chan CommitItem, ci.config.NumWorkers)

	for i := 1; i <= ci.config.NumWorkers; i++ {
		w := worker{
			id:    i,
			ch:    ci.queue,
			ci:    ci,
			items: make([]CommitItem, 0, ci.config.FlushLens),
		}
		w.run()
		ci.workers = append(ci.workers, &w)
	}
	ci.wg.Add(ci.config.NumWorkers)

	ci.ticker = time.NewTicker(ci.config.FlushInterval)
	go func() {
		ctx := context.Background()
		for {
			select {
			case <-ci.done:
				return
			case <-ci.ticker.C:
				if ci.config.DebugLogger != nil {
					ci.config.DebugLogger.Printf("[indexer] Auto-flushing workers after %s\n", ci.config.FlushInterval)
				}
				for _, w := range ci.workers {
					w.mu.Lock()
					if len(w.items) > 0 {
						if err := w.flush(ctx); err != nil {
							w.mu.Unlock()
							continue
						}
					}
					w.mu.Unlock()
				}
			}
		}
	}()
}

type worker struct {
	id    int
	ch    <-chan CommitItem
	mu    sync.Mutex
	ci    *committer
	items []CommitItem
}

func (w *worker) run() {
	go func() {
		ctx := context.Background()

		if w.ci.config.DebugLogger != nil {
			w.ci.config.DebugLogger.Printf("[worker-%03d] Started\n", w.id)
		}
		defer w.ci.wg.Done()

		for item := range w.ch {
			w.mu.Lock()

			if w.ci.config.DebugLogger != nil {
				w.ci.config.DebugLogger.Printf("[worker-%03d] Received item [%s:%s]\n", w.id, item.Key, item.Value)
			}

			w.items = append(w.items, item)
			if len(w.items) >= w.ci.config.FlushLens {
				if err := w.flush(ctx); err != nil {
					w.mu.Unlock()
					continue
				}
			}
			w.mu.Unlock()
		}
	}()
}

func (w *worker) flush(ctx context.Context) error {
	if len(w.items) < 1 {
		if w.ci.config.DebugLogger != nil {
			w.ci.config.DebugLogger.Printf("[worker-%03d] Flush: Buffer empty\n", w.id)
		}
		return nil
	}

	defer func() {
		w.items = w.items[:0]
	}()

	if w.ci.config.DebugLogger != nil {
		w.ci.config.DebugLogger.Printf("[worker-%03d] Flush: %s\n", w.id)
	}

	atomic.AddUint64(&w.ci.stats.numRequests, 1)
	err := committerPipeSet(w.items)
	if err != nil {
		atomic.AddUint64(&w.ci.stats.numFailed, uint64(len(w.items)))
		return fmt.Errorf("flush: %s", err)
	}

	return err
}

// PipeSet 批量set value
func committerPipeSet(items []CommitItem) (err error) {
	pipe := rdb.Pipeline()
	for _, item := range items {
		pipe.Set(item.Key, item.Value, item.ExTime)
	}

	_, err = pipe.Exec()
	defer pipe.Close()
	return
}
