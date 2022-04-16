package collector

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"

	"github.com/shiimaxx/metal/types"
)

type CPUCollector struct{}

func (c *CPUCollector) Collect(done <-chan struct{}, send chan<- types.Metrics) {
	var metrics types.Metrics
	ticker := time.NewTicker(time.Second)

	var previousCPUStats []cpu.TimesStat
	var latestCPUStats []cpu.TimesStat
	var err error

	// for countern
	latestCPUStats, err = cpu.TimesWithContext(context.TODO(), false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
	}

	for {
		select {
		case t := <-ticker.C:
			previousCPUStats = latestCPUStats
			latestCPUStats, err = cpu.TimesWithContext(context.TODO(), false)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err)
			}

			previous := previousCPUStats[0]
			latest := latestCPUStats[0]

			diff := cpu.TimesStat{
				CPU:       latest.CPU,
				User:      latest.User - previous.User,
				System:    latest.System - previous.System,
				Idle:      latest.Idle - previous.Idle,
				Nice:      latest.Nice - previous.Nice,
				Iowait:    latest.Iowait - previous.Iowait,
				Irq:       latest.Irq - previous.Irq,
				Softirq:   latest.Softirq - previous.Softirq,
				Steal:     latest.Steal - previous.Steal,
				Guest:     latest.Guest - previous.Guest,
				GuestNice: latest.GuestNice - previous.GuestNice,
			}

			metrics.Data = append(metrics.Data, types.Metric{
				Name:      "CPUUser",
				Timestamp: t,
				Value:     diff.User / diff.Total() * 100,
			})
			metrics.Data = append(metrics.Data, types.Metric{
				Name:      "CPUSystem",
				Timestamp: t,
				Value:     diff.System / diff.Total() * 100,
			})
			metrics.Data = append(metrics.Data, types.Metric{
				Name:      "CPUIdle",
				Timestamp: t,
				Value:     diff.Idle / diff.Total() * 100,
			})
			metrics.Data = append(metrics.Data, types.Metric{
				Name:      "CPUNice",
				Timestamp: t,
				Value:     diff.Nice / diff.Total() * 100,
			})
			metrics.Data = append(metrics.Data, types.Metric{
				Name:      "CPUIowait",
				Timestamp: t,
				Value:     diff.Iowait / diff.Total() * 100,
			})
			metrics.Data = append(metrics.Data, types.Metric{
				Name:      "CPUIrq",
				Timestamp: t,
				Value:     diff.Irq / diff.Total() * 100,
			})
			metrics.Data = append(metrics.Data, types.Metric{
				Name:      "CPUSoftirq",
				Timestamp: t,
				Value:     diff.Softirq / diff.Total() * 100,
			})
			metrics.Data = append(metrics.Data, types.Metric{
				Name:      "CPUSteal",
				Timestamp: t,
				Value:     diff.Steal / diff.Total() * 100,
			})
			metrics.Data = append(metrics.Data, types.Metric{
				Name:      "CPUGuest",
				Timestamp: t,
				Value:     diff.Guest / diff.Total() * 100,
			})
			metrics.Data = append(metrics.Data, types.Metric{
				Name:      "CPUGuestNice",
				Timestamp: t,
				Value:     diff.GuestNice / diff.Total() * 100,
			})
		case <-done:
			send <- metrics
			return
		}
	}
}
