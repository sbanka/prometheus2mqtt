package ticker

import (
	"context"
	"log"
	"time"

	"github.com/krzysztof-gzocha/prometheus2mqtt/config"
	"github.com/krzysztof-gzocha/prometheus2mqtt/publisher"
)

type Scraper interface {
	Scrape(ctx context.Context, metrics map[string]string) (map[string]string, error)
}

type Ticker struct {
	cfg       config.Config
	scraper   Scraper
	publisher publisher.Publisher
	logger    *log.Logger
}

func NewTicker(
	cfg config.Config,
	prometheus Scraper,
	publisher publisher.Publisher,
	logger *log.Logger,
) *Ticker {
	return &Ticker{
		cfg:       cfg,
		scraper:   prometheus,
		publisher: publisher,
		logger:    logger,
	}
}

func (t *Ticker) Start(ctx context.Context) {
	t.logger.Printf("Starting scraping for %d metric(s) every %s\n", len(t.cfg.Metrics), t.cfg.Interval.String())
	ticker := time.NewTicker(t.cfg.Interval)

	for {
		select {
		case <-ticker.C:
			t.tick(ctx)
		case <-ctx.Done():
			t.logger.Printf("Received signal to stop")
			ticker.Stop()
			return
		}
	}
}

func (t *Ticker) tick(ctx context.Context) {
	defer func() {
		e := recover()
		if e != nil {
			log.Printf("Recovering from panic: %+v\n", e)
		}
	}()

	ctxTimeout, cancel := context.WithTimeout(ctx, t.cfg.ScrapeTimeout)
	metrics, err := t.scraper.Scrape(ctxTimeout, t.cfg.Metrics)
	cancel()

	if err == context.DeadlineExceeded {
		t.logger.Printf("Scraping metrics exceeded timeout: %s\n", t.cfg.ScrapeTimeout.String())
	} else if err != nil {
		t.logger.Printf("Error when scraping for metrics: %s\n", err.Error())
	}

	for name, value := range metrics {
		err := t.publisher.Publish(ctx, name, value)
		if err != nil {
			t.logger.Printf("Error occurred when publishing metric %s: %s", name, err.Error())
		}
	}
}
