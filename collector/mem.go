package collector

import (
	"fmt"
	"os"
	"time"

	"github.com/shirou/gopsutil/v3/mem"

	"github.com/shiimaxx/metal/types"
)

type MemCollector struct{}

func (m *MemCollector) Collect(done <-chan struct{}, send chan<- types.Metrics) {
	var metrics types.Metrics
	ticker := time.NewTicker(time.Second)

	for {
		select {
		case t := <-ticker.C:
			memstat, err := mem.VirtualMemory()
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err)
			}

			metrics.Data = append(metrics.Data, types.Metric{
				Name:      "MemoryUsed",
				Timestamp: t,
				Value:     float64(memstat.Used),
			})
			metrics.Data = append(metrics.Data, types.Metric{
				Name:      "MemoryFree",
				Timestamp: t,
				Value:     float64(memstat.Free),
			})
			metrics.Data = append(metrics.Data, types.Metric{
				Name:      "MemoryShared",
				Timestamp: t,
				Value:     float64(memstat.Shared),
			})
			metrics.Data = append(metrics.Data, types.Metric{
				Name:      "MemoryBuffers",
				Timestamp: t,
				Value:     float64(memstat.Buffers),
			})
			metrics.Data = append(metrics.Data, types.Metric{
				Name:      "MemoryCached",
				Timestamp: t,
				Value:     float64(memstat.Cached),
			})
			metrics.Data = append(metrics.Data, types.Metric{
				Name:      "MemoryAvailable",
				Timestamp: t,
				Value:     float64(memstat.Available),
			})
		case <-done:
			send <- metrics
			return
		}
	}
}
