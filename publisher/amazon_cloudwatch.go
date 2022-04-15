package publisher

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/shiimaxx/istatsd/types"
)

type AmazonCloudWatchPublisher struct {
	Client *cloudwatch.Client
}

func (p *AmazonCloudWatchPublisher) Run(ctx context.Context, recieve <-chan types.Metrics) {
	for {
		select {
		case metrics := <-recieve:
			var mData []cwtypes.MetricDatum
			for _, m := range metrics.Data {
				dimensions := p.convertTags(m.Tags)
				mData = append(mData, cwtypes.MetricDatum{
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

func (p *AmazonCloudWatchPublisher) convertTags(tags map[string]string) []cwtypes.Dimension {
	var dimensions []cwtypes.Dimension

	for k, v := range tags {
		dimensions = append(dimensions, cwtypes.Dimension{
			Name:  aws.String(k),
			Value: aws.String(v),
		})
	}
	return dimensions
}
