package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/shiimaxx/metal/collector"
	"github.com/shiimaxx/metal/publisher"
	"github.com/shiimaxx/metal/types"
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
		var sec int
		var err error
		if paramSec := r.URL.Query().Get("seconds"); paramSec != "" {
			if sec, err = strconv.Atoi(paramSec); err != nil {
				http.Error(w, "seconds must be integer", http.StatusBadRequest)
				return
			}

			if sec < 10 {
				http.Error(w, "seconds must be over than 10", http.StatusBadRequest)
				return
			}
		} else {
			sec = 60
		}

		s.duration <- time.Second * time.Duration(sec)
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

	publisher := publisher.PublisherManager{
		Publishers: []publisher.Publisher{
			&publisher.StdoutPublisher{},
		},
	}
	go publisher.Run(ctx, metricCh)

	server := Server{
		port:     8080,
		duration: duration,
	}
	if err := server.run(ctx); err != nil {
		log.Fatal(err)
	}
}
