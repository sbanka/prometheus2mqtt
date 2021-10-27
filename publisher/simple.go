package publisher

import (
	"context"
	"fmt"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/krzysztof-gzocha/prometheus2mqtt/config"
)

type Publisher interface {
	Publish(ctx context.Context, name, value string) error
}

type Simple struct {
	cfg    config.Mqtt
	mqtt   mqtt.Client
	logger *log.Logger
}

func NewSimple(
	cfg config.Mqtt,
	mqtt mqtt.Client,
	logger *log.Logger,
) *Simple {
	return &Simple{
		cfg:    cfg,
		mqtt:   mqtt,
		logger: logger,
	}
}

func (s *Simple) Publish(ctx context.Context, name, value string) error {
	topic := s.cfg.PublishTopicPrefix + "/" + name
	token := s.mqtt.Publish(
		topic,
		s.cfg.Qos,
		s.cfg.RetainMessages,
		value,
	)

	ctx, cancel := context.WithTimeout(ctx, s.cfg.PublishTimeout)
	defer cancel()

	select {
	case <-token.Done():
		if token.Error() != nil {
			return token.Error()
		}
		s.logger.Printf("Sending \t%s\t to \t%s\n", value, topic)

		return nil
	case <-ctx.Done():
		return fmt.Errorf("publishing exceeded timeout: %w", token.Error())
	}
}
