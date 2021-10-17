package ticker

import (
	"context"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/krzysztof-gzocha/prometheus2mqtt/config"
)

type Scraper interface {
	Scrape(ctx context.Context, metrics map[string]string) (map[string]string, error)
}

type Ticker struct {
	cfg     config.Config
	scraper Scraper
	mqtt    mqtt.Client
	logger  *log.Logger
}

func NewTicker(
	cfg config.Config,
	prometheus Scraper,
	mqtt mqtt.Client,
	logger *log.Logger,
) *Ticker {
	return &Ticker{
		cfg:     cfg,
		scraper: prometheus,
		mqtt:    mqtt,
		logger:  logger,
	}
}

func (t *Ticker) Start(ctx context.Context) {
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
		topic := t.cfg.MqttBroker.PublishTopicPrefix + "/" + name
		token := t.mqtt.Publish(
			topic,
			t.cfg.MqttBroker.Qos,
			t.cfg.MqttBroker.RetainMessages,
			value,
		)

		if !token.WaitTimeout(t.cfg.MqttBroker.PublishTimeout) {
			t.logger.Printf(
				"Publishing metric %s took over %s, continuing without it...\n",
				name,
				t.cfg.MqttBroker.PublishTimeout.String(),
			)
			continue
		}

		if token.Error() != nil {
			t.logger.Printf("There was an error sending the message: %v", token.Error())
			continue
		}

		t.logger.Printf("Metric %s was send to MQTT topic %s with value:\t%s\n", name, topic, value)
	}
}
