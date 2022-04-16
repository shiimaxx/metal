package publisher

import (
	"context"

	"github.com/shiimaxx/metal/types"
)

type Publisher interface {
	Publish(context.Context, types.Metrics)
}

type PublisherManager struct {
	Publishers []Publisher
}

func (p *PublisherManager) Run(ctx context.Context, recieve <-chan types.Metrics) {
	for {
		select {
		case m := <-recieve:
			for _, p := range p.Publishers {
				p.Publish(ctx, m)
			}
		case <-ctx.Done():
			return
		}
	}
}
