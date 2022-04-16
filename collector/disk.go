package collector

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/shirou/gopsutil/v3/disk"

	"github.com/shiimaxx/metal/types"
)

type DiskIOCollector struct {
	Devices []string
}

func (d *DiskIOCollector) Collect(done <-chan struct{}, send chan<- types.Metrics) {
	var metrics types.Metrics
	ticker := time.NewTicker(time.Second)

	previous := make(map[string]disk.IOCountersStat)
	var latest map[string]disk.IOCountersStat
	var err error

	latest, err = disk.IOCountersWithContext(context.TODO(), d.Devices...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
	}

	for {
		select {
		case t := <-ticker.C:
			for k, v := range latest {
				previous[k] = v
			}
			latest, err = disk.IOCountersWithContext(context.TODO(), d.Devices...)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err)
			}

			for device, l := range latest {
				tags := map[string]string{
					"device": device,
				}

				p := previous[device]
				diff := disk.IOCountersStat{
					ReadCount:        l.ReadCount - p.ReadCount,
					MergedReadCount:  l.MergedReadCount - p.MergedReadCount,
					WriteCount:       l.WriteCount - p.WriteCount,
					MergedWriteCount: l.MergedWriteCount - p.MergedWriteCount,
					ReadBytes:        l.ReadBytes - p.ReadBytes,
					WriteBytes:       l.WriteBytes - p.WriteBytes,
					ReadTime:         l.ReadTime - p.ReadTime,
					WriteTime:        l.WriteTime - p.WriteTime,
					IopsInProgress:   l.IopsInProgress - p.IopsInProgress,
					IoTime:           l.IoTime - p.IoTime,
					WeightedIO:       l.WeightedIO - p.WeightedIO,
					Name:             l.Name,
					SerialNumber:     l.SerialNumber,
					Label:            l.Label,
				}
				metrics.Data = append(metrics.Data, types.Metric{
					Name:      "DiskReadCount",
					Timestamp: t,
					Value:     float64(diff.ReadCount),
					Tags:      tags,
				})
				metrics.Data = append(metrics.Data, types.Metric{
					Name:      "DiskWriteCount",
					Timestamp: t,
					Value:     float64(diff.WriteCount),
					Tags:      tags,
				})
				metrics.Data = append(metrics.Data, types.Metric{
					Name:      "DiskReadBytes",
					Timestamp: t,
					Value:     float64(diff.ReadBytes),
					Tags:      tags,
				})
				metrics.Data = append(metrics.Data, types.Metric{
					Name:      "DiskWriteBytes",
					Timestamp: t,
					Value:     float64(diff.WriteBytes),
					Tags:      tags,
				})
			}
		case <-done:
			send <- metrics
			return
		}
	}
}
