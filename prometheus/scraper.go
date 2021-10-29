package prometheus

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/krzysztof-gzocha/prometheus2mqtt/config"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type Scraper struct {
	prometheusClient v1.API
	logger           *log.Logger
}

func NewScraper(prometheus v1.API, logger *log.Logger) Scraper {
	return Scraper{prometheusClient: prometheus, logger: logger}
}

func (s Scraper) Scrape(ctx context.Context, metrics ...config.Metric) (map[string]string, error) {
	result := make(map[string]string)

	for _, metric := range metrics {
		val, _, err := s.prometheusClient.Query(ctx, metric.Query, time.Time{})
		if err != nil {
			return result, err
		}

		switch v := val.(type) {
		// @todo add all possible types coming from Query()
		case model.Vector:
			if len(v) == 0 {
				continue
			}
			result[metric.Name] = strconv.FormatFloat(
				float64(v[0].Value),
				'f',
				-1,
				64,
			)
		default:
			s.logger.Printf(
				"Metric %s is type %T, not model.Vector. Skipping it..",
				metric.Name,
				val,
			)
		}
	}

	return result, nil
}
