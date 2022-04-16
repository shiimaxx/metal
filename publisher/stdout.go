package publisher

import (
	"context"
	"fmt"

	"github.com/shiimaxx/metal/types"
)

type StdoutPublisher struct{}

func (p *StdoutPublisher) Publish(ctx context.Context, metrics types.Metrics) {
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
}
