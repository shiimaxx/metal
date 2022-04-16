package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shiimaxx/istatsd/collector"
	"github.com/shiimaxx/istatsd/publisher"
	"github.com/shiimaxx/istatsd/types"
)

type Server struct {
	port int

	duration chan<- time.Duration
}

func (s *Server) run(ctx context.Context) error {
	mux := http.DefaultServeMux
	mux.HandleFunc("/collect", s.collect())

	srv := &http.Server{
		Handler: mux,
	}

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return err
	}

	errCh := make(chan error)
	go func() {
		if err := srv.Serve(l); err != nil {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return nil
	}
}

func (s *Server) collect() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.duration <- time.Second * 10
	}
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
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

	// publisher, err := publisher.NewAmazonCloudEatchPublisher()
	// if err != nil {
	// 	log.Fatal(err)
	// }

	publisher := publisher.StdoutPublisher{}
	go publisher.Run(ctx, metricCh)

	server := Server{
		port:     8080,
		duration: duration,
	}
	if err := server.run(ctx); err != nil {
		log.Fatal(err)
	}
}
