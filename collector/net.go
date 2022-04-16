package collector

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/shirou/gopsutil/v3/net"

	"github.com/shiimaxx/metal/types"
)

type NetIOCollector struct {
}

func (n *NetIOCollector) Collect(done <-chan struct{}, send chan<- types.Metrics) {
	var metrics types.Metrics
	ticker := time.NewTicker(time.Second)

	var previous []net.IOCountersStat
	var latest []net.IOCountersStat
	var err error

	latest, err = net.IOCountersWithContext(context.TODO(), false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
	}

	for {
		select {
		case t := <-ticker.C:
			previous = latest
			latest, err = net.IOCountersWithContext(context.TODO(), false)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err)
			}

			p := previous[0]
			l := latest[0]

			diff := net.IOCountersStat{
				Name:        l.Name,
				BytesSent:   l.BytesSent - p.BytesSent,
				BytesRecv:   l.BytesRecv - p.BytesRecv,
				PacketsSent: l.PacketsSent - p.PacketsSent,
				PacketsRecv: l.PacketsRecv - p.PacketsRecv,
				Errin:       l.Errin - p.Errin,
				Errout:      l.Errout - p.Errout,
				Dropin:      l.Dropin - p.Dropin,
				Dropout:     l.Dropout - p.Dropout,
				Fifoin:      l.Fifoin - p.Fifoin,
				Fifoout:     l.Fifoout - p.Fifoout,
			}

			metrics.Data = append(metrics.Data, types.Metric{
				Name:      "NetworkBytesSent",
				Timestamp: t,
				Value:     float64(diff.BytesSent),
			})
			metrics.Data = append(metrics.Data, types.Metric{
				Name:      "NetworkBytesRecv",
				Timestamp: t,
				Value:     float64(diff.BytesRecv),
			})
			metrics.Data = append(metrics.Data, types.Metric{
				Name:      "NetworkPacketSent",
				Timestamp: t,
				Value:     float64(diff.PacketsSent),
			})
			metrics.Data = append(metrics.Data, types.Metric{
				Name:      "NetworkPacketRecv",
				Timestamp: t,
				Value:     float64(diff.PacketsRecv),
			})
		case <-done:
			send <- metrics
			return
		}
	}
}
