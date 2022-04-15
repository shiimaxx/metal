package publisher

import (
	"context"

	"github.com/shiimaxx/istatsd/types"
)

type Publisher interface {
	Run(context.Context, <-chan []types.Metric)
}
