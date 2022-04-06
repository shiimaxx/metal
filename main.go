package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
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

	_, err := cpu.Percent(0, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
	}

	for {
		select {
		case t := <-ticker.C:
			cpuuser, err := cpu.Percent(0, false)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err)
			}

			metrics = append(metrics, Metric{
				Name:      "cpu",
				Timestamp: t,
				Value:     cpuuser[0],
			})
		case <-done:
			send <- metrics
			return
		}
	}
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

	for range c.Collectors {
		send <- <-recieve
	}
}

type Publisher interface {
	Run(context.Context, <-chan []Metric)
}

type StdoutPublisher struct{}

func (p *StdoutPublisher) Run(ctx context.Context, recieve <-chan []Metric) {
	for {
		for _, metric := range <-recieve {
			fmt.Printf("%+v\n", metric)
		}
	}
}

type AmazonCloudWatchPublisher struct {
	Client *cloudwatch.Client
}

func (p *AmazonCloudWatchPublisher) Run(ctx context.Context, recieve <-chan []Metric) {
	for {
		for _, metric := range <-recieve {
			input := &cloudwatch.PutMetricDataInput{
				MetricData: []types.MetricDatum{
					{
						MetricName: aws.String(metric.Name),
						Timestamp:  &metric.Timestamp,
						Value:      &metric.Value,
					},
				},
				Namespace: aws.String("isd"),
			}
			_, err := p.Client.PutMetricData(ctx, input)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err)
			}
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

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("ap-northeast-1"))
	if err != nil {
		log.Fatal(err)
	}

	client := cloudwatch.NewFromConfig(cfg)

	publisher := AmazonCloudWatchPublisher{Client: client}
	go publisher.Run(ctx, metricCh)

	<-ctx.Done()
}
