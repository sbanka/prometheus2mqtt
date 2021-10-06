package prometheus

import (
	"context"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type Client struct {
	prometheusClient v1.API
}

func NewClient(prometheus v1.API) Client {
	return Client{prometheusClient: prometheus}
}

func (c Client) Scrape(ctx context.Context, metrics ...string) (Metrics, error) {
	result := make(Metrics, 0)

	for _, metric := range metrics {
		val, _, err := c.prometheusClient.Query(ctx, metric, time.Time{})
		if err != nil {
			return result, err
		}

		if vec, ok := val.(model.Vector); ok {
			for _, sample := range vec {
				result[sample.Metric.String()] = Metric{
					Value: float64(sample.Value),
					Time:  time.Unix(sample.Timestamp.Unix(), 0),
				}
			}
		}
	}

	return result, nil
}
