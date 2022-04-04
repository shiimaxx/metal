package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/mem"
)

type Metric struct {
	Name      string
	Timestamp time.Time
	Value     float64
}

type Collector interface {
	Collect(<-chan time.Time) []Metric
}

type CPUCollector struct{}

func (c *CPUCollector) Collect(timestampCh <-chan time.Time) []Metric {
	var metrics []Metric
	for t := range timestampCh {
		metrics = append(metrics, Metric{
			Name:      "cpu",
			Timestamp: t,
			Value:     c.userCPU(),
		})
	}
	return metrics
}

func (c *CPUCollector) userCPU() float64 {
	// cpustat, err := cpu.TimesWithContext(context.TODO(), false)
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "%s\n", err)
	// }
	// cpustat0 := cpustat[0]
	// return float64(cpustat0.User / cpustat0.Total())
	return 0.123456789
}

type MemCollector struct{}

func (m *MemCollector) Collect(timestampCh <-chan time.Time) []Metric {
	var metrics []Metric
	for t := range timestampCh {
		memstat, err := mem.VirtualMemory()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}

		metrics = append(metrics, Metric{
			Name:      "memory",
			Timestamp: t,
			Value:     memstat.UsedPercent,
		})
	}
	return metrics
}

type CollectorManager struct {
	Metrics []Metric
}

func (c *CollectorManager) Run(ctx context.Context, send chan<- []Metric) {
	ticker := c.Ticker(time.Second, 10)

	cpuCollector := CPUCollector{}
	memCollector := MemCollector{}

	var wg sync.WaitGroup
	wg.Add(2)

	var cpuMetrics []Metric
	go func() {
		defer wg.Done()
		cpuMetrics = cpuCollector.Collect(ticker)
	}()

	var memMetrics []Metric
	go func() {
		defer wg.Done()
		memMetrics = memCollector.Collect(ticker)
	}()

	wg.Wait()

	var metrics []Metric
	metrics = append(metrics, cpuMetrics...)
	metrics = append(metrics, memMetrics...)
	send <- metrics
}

func (c *CollectorManager) Ticker(d time.Duration, count int) chan time.Time {
	ticker := make(chan time.Time)
	internalTicker := time.NewTicker(d)

	go func() {
		i := 0
		for c := range internalTicker.C {
			ticker <- c
			if i >= count {
				internalTicker.Stop()
				close(ticker)
				break
			}
			i++
		}
	}()

	return ticker
}

type Publisher struct{}

func (p *Publisher) Run(ctx context.Context, recieve <-chan []Metric) {
	for {
		for _, metric := range <-recieve {
			fmt.Printf("%+v\n", metric)
		}
	}
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	metricCh := make(chan []Metric)

	collector := CollectorManager{}
	go collector.Run(ctx, metricCh)

	publisher := Publisher{}
	go publisher.Run(ctx, metricCh)

	<-ctx.Done()
}
