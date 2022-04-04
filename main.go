package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

type Metric struct {
	Name      string
	Timestamp time.Time
	Value     float64
}

type Collector interface {
	Collect(<-chan time.Time, chan<- Metric)
}

type CPUCollector struct{}

func (c *CPUCollector) Collect(timestampCh <-chan time.Time, metricCh chan<- Metric) {
	for {
		timestamp := <-timestampCh

		metricCh <- Metric{
			Name:      "cpu",
			Timestamp: timestamp,
			Value:     c.userCPU(),
		}
	}
}

func (c *CPUCollector) userCPU() float64 {
	cpustat, err := cpu.TimesWithContext(context.TODO(), false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
	}
	cpustat0 := cpustat[0]

	return float64(cpustat0.User / cpustat0.Total())
}

type MemCollector struct{}

func (m *MemCollector) Collect(timestampCh <-chan time.Time, metricCh chan<- Metric) {
	for {
		timestamp := <-timestampCh

		memstat, err := mem.VirtualMemory()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}

		metricCh <- Metric{
			Name:      "memory",
			Timestamp: timestamp,
			Value:     memstat.UsedPercent,
		}
	}
}

type CollectorTODORename struct {
	Metrics []Metric
}

func (c *CollectorTODORename) Run(ctx context.Context, tick <-chan time.Time, send chan<- []Metric) {
	metricCh := make(chan Metric)

	cpuCollector := CPUCollector{}
	memCollector := MemCollector{}

	go cpuCollector.Collect(tick, metricCh)
	go memCollector.Collect(tick, metricCh)

	// var metrics []Metric
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

	publisher := Publisher{}
	go publisher.Run(ctx, metricCh)

	<-ctx.Done()
}
