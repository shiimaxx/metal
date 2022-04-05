package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/shirou/gopsutil/v3/mem"
)

type Metric struct {
	Name      string
	Timestamp time.Time
	Value     float64
}

type Collector interface {
	Collect(<-chan struct{}, chan<- []Metric)
}

type CPUCollector struct{}

func (c *CPUCollector) Collect(done <-chan struct{}, send chan<- []Metric) {
	var metrics []Metric
	ticker := time.NewTicker(time.Second)

	for {
		select {
		case t := <-ticker.C:
			metrics = append(metrics, Metric{
				Name:      "cpu",
				Timestamp: t,
				Value:     c.userCPU(),
			})
		case <-done:
			send <- metrics
			return
		}
	}
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

func (m *MemCollector) Collect(done <-chan struct{}, send chan<- []Metric) {
	var metrics []Metric
	ticker := time.NewTicker(time.Second)

	for {
		select {
		case t := <-ticker.C:
			memstat, err := mem.VirtualMemory()
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err)
			}

			metrics = append(metrics, Metric{
				Name:      "memory",
				Timestamp: t,
				Value:     memstat.UsedPercent,
			})
		case <-done:
			send <- metrics
			return
		}
	}
}

type CollectorManager struct {
	Collectors []Collector
}

func (c *CollectorManager) Run(ctx context.Context, send chan<- []Metric) {
	done := make(chan struct{})
	recieve := make(chan []Metric)

	for _, collector := range c.Collectors {
		go collector.Collect(done, recieve)
	}

	time.Sleep(time.Second * 10)
	close(done)

	var metrics []Metric
	for range c.Collectors {
		m := <-recieve
		metrics = append(metrics, m...)
	}
	send <- metrics
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

	collector := CollectorManager{
		Collectors: []Collector{
			&CPUCollector{}, &MemCollector{},
		},
	}
	go collector.Run(ctx, metricCh)

	publisher := Publisher{}
	go publisher.Run(ctx, metricCh)

	<-ctx.Done()
}
