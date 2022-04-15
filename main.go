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
	"github.com/shirou/gopsutil/v3/net"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

type Metrics struct {
	Data []Metric
}

type Metric struct {
	Name      string
	Timestamp time.Time
	Value     float64
	Tags      map[string]string
}

type Collector interface {
	Collect(<-chan struct{}, chan<- Metrics)
}

type CPUCollector struct{}

func (c *CPUCollector) Collect(done <-chan struct{}, send chan<- Metrics) {
	var metrics Metrics
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

			metrics.Data = append(metrics.Data, Metric{
				Name:      "CPUUser",
				Timestamp: t,
				Value:     diff.User / diff.Total() * 100,
			})
			metrics.Data = append(metrics.Data, Metric{
				Name:      "CPUSystem",
				Timestamp: t,
				Value:     diff.System / diff.Total() * 100,
			})
			metrics.Data = append(metrics.Data, Metric{
				Name:      "CPUIdle",
				Timestamp: t,
				Value:     diff.Idle / diff.Total() * 100,
			})
			metrics.Data = append(metrics.Data, Metric{
				Name:      "CPUNice",
				Timestamp: t,
				Value:     diff.Nice / diff.Total() * 100,
			})
			metrics.Data = append(metrics.Data, Metric{
				Name:      "CPUIowait",
				Timestamp: t,
				Value:     diff.Iowait / diff.Total() * 100,
			})
			metrics.Data = append(metrics.Data, Metric{
				Name:      "CPUIrq",
				Timestamp: t,
				Value:     diff.Irq / diff.Total() * 100,
			})
			metrics.Data = append(metrics.Data, Metric{
				Name:      "CPUSoftirq",
				Timestamp: t,
				Value:     diff.Softirq / diff.Total() * 100,
			})
			metrics.Data = append(metrics.Data, Metric{
				Name:      "CPUSteal",
				Timestamp: t,
				Value:     diff.Steal / diff.Total() * 100,
			})
			metrics.Data = append(metrics.Data, Metric{
				Name:      "CPUGuest",
				Timestamp: t,
				Value:     diff.Guest / diff.Total() * 100,
			})
			metrics.Data = append(metrics.Data, Metric{
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

func (m *MemCollector) Collect(done <-chan struct{}, send chan<- Metrics) {
	var metrics Metrics
	ticker := time.NewTicker(time.Second)

	for {
		select {
		case t := <-ticker.C:
			memstat, err := mem.VirtualMemory()
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err)
			}

			metrics.Data = append(metrics.Data, Metric{
				Name:      "MemoryUsed",
				Timestamp: t,
				Value:     float64(memstat.Used),
			})
			metrics.Data = append(metrics.Data, Metric{
				Name:      "MemoryFree",
				Timestamp: t,
				Value:     float64(memstat.Free),
			})
			metrics.Data = append(metrics.Data, Metric{
				Name:      "MemoryShared",
				Timestamp: t,
				Value:     float64(memstat.Shared),
			})
			metrics.Data = append(metrics.Data, Metric{
				Name:      "MemoryBuffers",
				Timestamp: t,
				Value:     float64(memstat.Buffers),
			})
			metrics.Data = append(metrics.Data, Metric{
				Name:      "MemoryCached",
				Timestamp: t,
				Value:     float64(memstat.Cached),
			})
			metrics.Data = append(metrics.Data, Metric{
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

func (d *DiskIOCollector) Collect(done <-chan struct{}, send chan<- Metrics) {
	var metrics Metrics
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
				metrics.Data = append(metrics.Data, Metric{
					Name:      "DiskReadCount",
					Timestamp: t,
					Value:     float64(diff.ReadCount),
					Tags:      tags,
				})
				metrics.Data = append(metrics.Data, Metric{
					Name:      "DiskWriteCount",
					Timestamp: t,
					Value:     float64(diff.WriteCount),
					Tags:      tags,
				})
				metrics.Data = append(metrics.Data, Metric{
					Name:      "DiskReadBytes",
					Timestamp: t,
					Value:     float64(diff.ReadBytes),
					Tags:      tags,
				})
				metrics.Data = append(metrics.Data, Metric{
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

type NetIOCollector struct {
}

func (n *NetIOCollector) Collect(done <-chan struct{}, send chan<- Metrics) {
	var metrics Metrics
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

			metrics.Data = append(metrics.Data, Metric{
				Name:      "NetworkBytesSent",
				Timestamp: t,
				Value:     float64(diff.BytesSent),
			})
			metrics.Data = append(metrics.Data, Metric{
				Name:      "NetworkBytesRecv",
				Timestamp: t,
				Value:     float64(diff.BytesRecv),
			})
			metrics.Data = append(metrics.Data, Metric{
				Name:      "NetworkPacketSent",
				Timestamp: t,
				Value:     float64(diff.PacketsSent),
			})
			metrics.Data = append(metrics.Data, Metric{
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

type CollectorManager struct {
	Collectors []Collector
}

func (c *CollectorManager) Run(ctx context.Context, duration <-chan time.Duration, send chan<- Metrics) {
	for {
		select {
		case d := <-duration:
			done := make(chan struct{})
			recieve := make(chan Metrics)

			for _, collector := range c.Collectors {
				go collector.Collect(done, recieve)
			}

			time.Sleep(d)
			close(done)

			for range c.Collectors {
				send <- <-recieve
			}
		case <-ctx.Done():
			return
		}
	}
}

type Publisher interface {
	Run(context.Context, <-chan []Metric)
}

type StdoutPublisher struct{}

func (p *StdoutPublisher) Run(ctx context.Context, recieve <-chan Metrics) {
	for {
		select {
		case metrics := <-recieve:
			for _, m := range metrics.Data {
				var tags string
				if len(m.Tags) > 0 {
					tags += "{"
					for k, v := range m.Tags {
						tags += fmt.Sprintf("%s=\"%s\"", k, v)
					}
					tags += "}"
				}
				fmt.Printf("%s%s %f %d\n", m.Name, tags, m.Value, m.Timestamp.Unix())
			}
		case <-ctx.Done():
			return
		}
	}
}

type AmazonCloudWatchPublisher struct {
	Client *cloudwatch.Client
}

func (p *AmazonCloudWatchPublisher) Run(ctx context.Context, recieve <-chan Metrics) {
	for {
		select {
		case metrics := <-recieve:
			var mData []types.MetricDatum
			for _, m := range metrics.Data {
				dimensions := p.convertTags(m.Tags)
				mData = append(mData, types.MetricDatum{
					MetricName:        aws.String(m.Name),
					Timestamp:         &m.Timestamp,
					Value:             &m.Value,
					StorageResolution: aws.Int32(1),
					Dimensions:        dimensions,
				})
			}
			input := &cloudwatch.PutMetricDataInput{
				MetricData: mData,
				Namespace:  aws.String("IStatsD"),
			}
			_, err := p.Client.PutMetricData(ctx, input)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err)
			}
		case <-ctx.Done():
			return
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

	duration := make(chan time.Duration)
	metricCh := make(chan Metrics)

	collector := CollectorManager{
		Collectors: []Collector{
			&CPUCollector{},
			&MemCollector{},
			// &DiskIOCollector{Devices: []string{"sda"}},
		},
	}
	go collector.Run(ctx, duration, metricCh)

	// cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("ap-northeast-1"))
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// client := cloudwatch.NewFromConfig(cfg)
	// publisher := AmazonCloudWatchPublisher{Client: client}

	publisher := StdoutPublisher{}
	go publisher.Run(ctx, metricCh)

	duration <- time.Second * 10

	<-ctx.Done()
}
