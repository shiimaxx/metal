package collector

import (
	"context"
	"time"

	"github.com/shiimaxx/istatsd/types"
)

type Collector interface {
	Collect(<-chan struct{}, chan<- types.Metrics)
}

type CollectorManager struct {
	Collectors []Collector
}

func (c *CollectorManager) Run(ctx context.Context, duration <-chan time.Duration, send chan<- types.Metrics) {
	for {
		select {
		case d := <-duration:
			done := make(chan struct{})
			recieve := make(chan types.Metrics)

			for _, collector := range c.Collectors {
				go collector.Collect(done, recieve)
			}

			time.Sleep(d)
			close(done)

			for range c.Collectors {
				send <- <-recieve
			}
		case <-ctx.Done():
			return
		}
	}
}
