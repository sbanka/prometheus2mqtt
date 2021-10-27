package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"log"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/krzysztof-gzocha/prometheus2mqtt/config"
)

const deviceName = "prometheus2mqtt"
const deviceManufacturer = "Krzysztof Gzocha Twitter:@kgzocha"

type haConfigMessage struct {
	Name       string   `json:"name"`
	StateTopic string   `json:"state_topic"`
	Device     haDevice `json:"device"`
}

type haDevice struct {
	Manufacturer string `json:"manufacturer"`
	Name         string `json:"name"`
	Identifiers  string `json:"identifiers"`
	Version      string `json:"sw_version"`
}

type HomeAssistant struct {
	cfg               config.Mqtt
	mqtt              mqtt.Client
	logger            *log.Logger
	alreadyConfigured map[string]interface{}
}

func NewHomeAssistant(
	cfg config.Mqtt,
	mqtt mqtt.Client,
	logger *log.Logger,
) *HomeAssistant {
	return &HomeAssistant{
		cfg:               cfg,
		mqtt:              mqtt,
		logger:            logger,
		alreadyConfigured: make(map[string]interface{}),
	}
}

func (h *HomeAssistant) Publish(ctx context.Context, name, value string) error {
	if !h.isConfigured(name) {
		err := h.configure(ctx, name)
		if err != nil {
			return fmt.Errorf("could not send configuration message for metric %s: %s", name, err.Error())
		}
	}

	h.logger.Printf("Sending \t%s\t to \t%s\n", value, h.stateTopic(name))

	return h.sendMsg(ctx, h.stateTopic(name), value)
}

func (h *HomeAssistant) isConfigured(name string) bool {
	_, exists := h.alreadyConfigured[name]

	return exists
}

func (h *HomeAssistant) configure(ctx context.Context, name string) error {
	sensorName := h.sensorName(name)
	h.logger.Printf("Configuring sensor: %s (ID: %s)\n", sensorName, shortHash(sensorName))

	haCfg := haConfigMessage{
		Name:       sensorName,
		StateTopic: h.stateTopic(name),
		Device: haDevice{
			Manufacturer: deviceManufacturer,
			Name:         deviceName,
			Version:      config.Version,
			Identifiers:  shortHash(sensorName),
		},
	}

	j, err := json.Marshal(&haCfg)
	if err != nil {
		return err
	}

	h.logger.Printf(
		"Configuring device on topic %s with payload %s\n",
		h.configTopic(name),
		string(j),
	)

	err = h.sendMsg(ctx, h.configTopic(name), string(j))
	if err != nil {
		return err
	}

	h.alreadyConfigured[name] = struct{}{}

	return nil
}

func (h *HomeAssistant) sensorName(name string) string {
	return h.cfg.ClientID + ": " + name
}

func (h *HomeAssistant) stateTopic(name string) string {
	return fmt.Sprintf(
		"%s/sensor/%s/state",
		h.cfg.DiscoveryPrefix,
		h.stripSlashes(h.cfg.ClientID+"_"+name),
	)
}

func (h *HomeAssistant) configTopic(name string) string {
	return fmt.Sprintf(
		"%s/sensor/%s/config",
		h.cfg.DiscoveryPrefix,
		h.stripSlashes(h.cfg.ClientID+"_"+name),
	)
}

func (h *HomeAssistant) stripSlashes(name string) string {
	return strings.Replace(name, "/", "_", -1)
}

func (h *HomeAssistant) sendMsg(ctx context.Context, topic, msg string) error {
	token := h.mqtt.Publish(
		topic,
		h.cfg.Qos,
		h.cfg.RetainMessages,
		msg,
	)

	ctx, cancel := context.WithTimeout(ctx, h.cfg.PublishTimeout)
	defer cancel()

	select {
	case <-token.Done():
		return token.Error()
	case <-ctx.Done():
		return fmt.Errorf("publishing exceeded timeout: %w", token.Error())
	}
}

func shortHash(input string) string {
	h := crc32.NewIEEE() //nolint:gosec
	_, _ = h.Write([]byte(input))

	return string(h.Sum(nil))
}
