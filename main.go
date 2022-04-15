package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/shiimaxx/istatsd/collector"
	"github.com/shiimaxx/istatsd/publisher"
	"github.com/shiimaxx/istatsd/types"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	duration := make(chan time.Duration)
	metricCh := make(chan types.Metrics)

	collector := collector.CollectorManager{
		Collectors: []collector.Collector{
			&collector.CPUCollector{},
			&collector.MemCollector{},
			// &collector.DiskIOCollector{Devices: []string{"sda"}},
		},
	}
	go collector.Run(ctx, duration, metricCh)

	// cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("ap-northeast-1"))
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// client := cloudwatch.NewFromConfig(cfg)
	// publisher := publisher.AmazonCloudWatchPublisher{Client: client}

	publisher := publisher.StdoutPublisher{}
	go publisher.Run(ctx, metricCh)

	duration <- time.Second * 10

	<-ctx.Done()
}
