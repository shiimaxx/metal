package publisher

import (
	"context"
	"fmt"

	"github.com/shiimaxx/istatsd/types"
)

type StdoutPublisher struct{}

func (p *StdoutPublisher) Run(ctx context.Context, recieve <-chan types.Metrics) {
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
