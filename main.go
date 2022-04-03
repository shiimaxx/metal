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

type Collector struct {
	Metrics []Metric
}

func (c *Collector) Run(ctx context.Context, send chan<- []Metric) {
	for i := 0; i < 10; i++ {
		timestamp := time.Now()

		cpustat, err := cpu.PercentWithContext(ctx, time.Second, false)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}

		memstat, err := mem.VirtualMemory()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}

		c.Metrics = append(c.Metrics, Metric{
			Name:      "cpu.user",
			Timestamp: timestamp,
			Value:     cpustat[0],
		})

		c.Metrics = append(c.Metrics, Metric{
			Name:      "memory",
			Timestamp: timestamp,
			Value:     memstat.UsedPercent,
		})
	}
	send <- c.Metrics
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

	collector := Collector{}
	go collector.Run(ctx, metricCh)

	publisher := Publisher{}
	go publisher.Run(ctx, metricCh)

	<-ctx.Done()
}
