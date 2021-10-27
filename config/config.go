package config

import (
	"net/url"
	"os"
	"time"

	"github.com/kelseyhightower/envconfig"
)

var Version string

type Config struct {
	PrometheusUrl string            `envconfig:"prometheus_url" required:"true"`
	Mqtt          Mqtt              `envconfig:"mqtt"`
	Metrics       map[string]string `envconfig:"metrics" default:"disks_flushes:node_disk_flush_requests_total{device='sda'}"`
	Interval      time.Duration     `envconfig:"interval" default:"15s"`
	ScrapeTimeout time.Duration     `envconfig:"scrape_timeout" default:"3s"`
}

type Mqtt struct {
	User               string        `envconfig:"user"`
	Password           string        `envconfig:"password"`
	UserFile           string        `envconfig:"user_file"`
	PasswordFile       string        `envconfig:"password_file"`
	Servers            []string      `envconfig:"servers" required:"true"`
	InsecureSkipVerify bool          `envconfig:"insecure_skip_verify" default:"false"`
	PublishTopicPrefix string        `envconfig:"publish_topic_prefix" default:"p2m"`
	ClientID           string        `envconfig:"client_id" default:"Prometheus2MQTT"`
	RetainMessages     bool          `envconfig:"retain_messages" default:"true"`
	PublishTimeout     time.Duration `envconfig:"publish_timeout" default:"5s"`
	Qos                byte          `envconfig:"qos" default:"1"`
	// HAPublisher defines if MQTT publishing format should be compatible with HomeAssistant
	HAPublisher     bool   `envconfig:"ha_publisher" default:"true"`
	DiscoveryPrefix string `envconfig:"discovery_prefix" default:"homeassistant"`
}

func (m Mqtt) ServersUrls() ([]*url.URL, error) {
	mqttServers := make([]*url.URL, 0)
	for _, server := range m.Servers {
		serverUrl, err := url.Parse(server)
		if err != nil {
			return nil, err
		}
		mqttServers = append(mqttServers, serverUrl)
	}

	return mqttServers, nil
}

// GetUser will return the correct User value
func (m Mqtt) GetUser() string {
	if m.UserFile == "" {
		return m.User
	}

	user, err := os.ReadFile(m.UserFile)
	if err != nil {
		panic(err.Error())
	}

	return string(user[0 : len(user)-1])
}

func (m Mqtt) GetPassword() string {
	if m.PasswordFile == "" {
		return m.User
	}

	password, err := os.ReadFile(m.PasswordFile)
	if err != nil {
		panic(err.Error())
	}

	return string(password[0 : len(password)-1])
}

func Load() (Config, error) {
	c := Config{}
	err := envconfig.Process("p2m", &c)

	return c, err
}
