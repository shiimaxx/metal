package publisher

import (
	"context"
	"fmt"

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
		fmt.Printf("dubug1: %s %f %s\n", m.Name, m.Value, m.Timestamp)
		dimensions := p.convertTags(m.Tags)
		data := cwtypes.MetricDatum{
			MetricName:        aws.String(m.Name),
			Timestamp:         aws.Time(m.Timestamp),
			Value:             aws.Float64(m.Value),
			StorageResolution: aws.Int32(1),
			Dimensions:        dimensions,
		}
		fmt.Printf("dubug2: %s %f %s\n", *data.MetricName, *data.Value, data.Timestamp)
		mData = append(mData, data)
		for _, d := range mData {
			fmt.Printf("dubug3: %s %f %s\n", *d.MetricName, *d.Value, d.Timestamp)
		}
	}
	for _, d := range mData {
		fmt.Printf("dubug4: %s %f %s\n", *d.MetricName, *d.Value, d.Timestamp)
	}
	// input := &cloudwatch.PutMetricDataInput{
	// 	MetricData: mData,
	// 	Namespace:  aws.String("metal"),
	// }
	// _, err := p.Client.PutMetricData(ctx, input)
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "%s\n", err)
	// }
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
