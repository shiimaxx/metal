package publisher

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	"github.com/shiimaxx/metal/types"
)

type AmazonCloudWatchPublisher struct {
	Client *cloudwatch.Client
}

func NewAmazonCloudEatchPublisher() (*AmazonCloudWatchPublisher, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("ap-northeast-1"))
	if err != nil {
		return nil, err
	}
	client := cloudwatch.NewFromConfig(cfg)
	return &AmazonCloudWatchPublisher{Client: client}, nil
}

func (p *AmazonCloudWatchPublisher) Publish(ctx context.Context, metrics types.Metrics) {
	var mData []cwtypes.MetricDatum
	for _, m := range metrics.Data {
		dimensions := p.convertTags(m.Tags)
		data := cwtypes.MetricDatum{
			MetricName:        aws.String(m.Name),
			Timestamp:         aws.Time(m.Timestamp),
			Value:             aws.Float64(m.Value),
			StorageResolution: aws.Int32(1),
			Dimensions:        dimensions,
		}
		mData = append(mData, data)
	}
	input := &cloudwatch.PutMetricDataInput{
		MetricData: mData,
		Namespace:  aws.String("METAL"),
	}
	_, err := p.Client.PutMetricData(ctx, input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
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
