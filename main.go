package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

type Metric struct {
	Name      string
	Timestamp time.Time
	Value     float64
	Tags      map[string]string
}

type Collector interface {
	Collect(<-chan struct{}, chan<- []Metric)
}

type CPUCollector struct{}

func (c *CPUCollector) Collect(done <-chan struct{}, send chan<- []Metric) {
	var metrics []Metric
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

			metrics = append(metrics, Metric{
				Name:      "CPUUser",
				Timestamp: t,
				Value:     diff.User / diff.Total() * 100,
			})
			metrics = append(metrics, Metric{
				Name:      "CPUSystem",
				Timestamp: t,
				Value:     diff.System / diff.Total() * 100,
			})
			metrics = append(metrics, Metric{
				Name:      "CPUIdle",
				Timestamp: t,
				Value:     diff.Idle / diff.Total() * 100,
			})
			metrics = append(metrics, Metric{
				Name:      "CPUNice",
				Timestamp: t,
				Value:     diff.Nice / diff.Total() * 100,
			})
			metrics = append(metrics, Metric{
				Name:      "CPUIowait",
				Timestamp: t,
				Value:     diff.Iowait / diff.Total() * 100,
			})
			metrics = append(metrics, Metric{
				Name:      "CPUIrq",
				Timestamp: t,
				Value:     diff.Irq / diff.Total() * 100,
			})
			metrics = append(metrics, Metric{
				Name:      "CPUSoftirq",
				Timestamp: t,
				Value:     diff.Softirq / diff.Total() * 100,
			})
			metrics = append(metrics, Metric{
				Name:      "CPUSteal",
				Timestamp: t,
				Value:     diff.Steal / diff.Total() * 100,
			})
			metrics = append(metrics, Metric{
				Name:      "CPUGuest",
				Timestamp: t,
				Value:     diff.Guest / diff.Total() * 100,
			})
			metrics = append(metrics, Metric{
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
				Name:      "MemoryUsed",
				Timestamp: t,
				Value:     float64(memstat.Used),
			})
			metrics = append(metrics, Metric{
				Name:      "MemoryFree",
				Timestamp: t,
				Value:     float64(memstat.Free),
			})
			metrics = append(metrics, Metric{
				Name:      "MemoryShared",
				Timestamp: t,
				Value:     float64(memstat.Shared),
			})
			metrics = append(metrics, Metric{
				Name:      "MemoryBuffers",
				Timestamp: t,
				Value:     float64(memstat.Buffers),
			})
			metrics = append(metrics, Metric{
				Name:      "MemoryCached",
				Timestamp: t,
				Value:     float64(memstat.Cached),
			})
			metrics = append(metrics, Metric{
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

type DiskIOCollector struct {
	Devices []string
}

func (d *DiskIOCollector) Collect(done <-chan struct{}, send chan<- []Metric) {
	var metrics []Metric
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
				metrics = append(metrics, Metric{
					Name:      "ReadCount",
					Timestamp: t,
					Value:     float64(diff.ReadCount),
					Tags:      tags,
				})
				metrics = append(metrics, Metric{
					Name:      "WriteCount",
					Timestamp: t,
					Value:     float64(diff.WriteCount),
					Tags:      tags,
				})
				metrics = append(metrics, Metric{
					Name:      "ReadBytes",
					Timestamp: t,
					Value:     float64(diff.ReadBytes),
					Tags:      tags,
				})
				metrics = append(metrics, Metric{
					Name:      "WriteBytes",
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
			var tags string
			if len(metric.Tags) > 0 {
				tags += "{"
				for k, v := range metric.Tags {
					tags += fmt.Sprintf("%s=\"%s\"", k, v)
				}
				tags += "}"
			}
			fmt.Printf("%s%s %f %d\n", metric.Name, tags, metric.Value, metric.Timestamp.Unix())
		}
	}
}

type AmazonCloudWatchPublisher struct {
	Client *cloudwatch.Client
}

func (p *AmazonCloudWatchPublisher) Run(ctx context.Context, recieve <-chan []Metric) {
	for {
		for _, metric := range <-recieve {
			dimensions := p.convertTags(metric.Tags)
			input := &cloudwatch.PutMetricDataInput{
				MetricData: []types.MetricDatum{
					{
						MetricName:        aws.String(metric.Name),
						Timestamp:         &metric.Timestamp,
						Value:             &metric.Value,
						StorageResolution: aws.Int32(1),
						Dimensions:        dimensions,
					},
				},
				Namespace: aws.String("IStatsD"),
			}
			_, err := p.Client.PutMetricData(ctx, input)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err)
			}
		}
	}
}

func (p *AmazonCloudWatchPublisher) convertTags(tags map[string]string) []types.Dimension {
	var dimensions []types.Dimension

	for k, v := range tags {
		dimensions = append(dimensions, types.Dimension{
			Name:  aws.String(k),
			Value: aws.String(v),
		})
	}
	return dimensions
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	metricCh := make(chan []Metric)

	collector := CollectorManager{
		Collectors: []Collector{
			&CPUCollector{},
			&MemCollector{},
			&DiskIOCollector{Devices: []string{"sda"}},
		},
	}
	go collector.Run(ctx, metricCh)

	// cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("ap-northeast-1"))
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// client := cloudwatch.NewFromConfig(cfg)
	// publisher := AmazonCloudWatchPublisher{Client: client}

	publisher := StdoutPublisher{}
	go publisher.Run(ctx, metricCh)

	<-ctx.Done()
}
