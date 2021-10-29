package config

import (
	"net/url"
	"os"
	"time"

	"github.com/spf13/viper"
)

var Version string

type Metric struct {
	Name  string `mapstructure:"name"`
	Query string `mapstructure:"query"`
}

type Config struct {
	PrometheusUrl string        `mapstructure:"prometheus_url" envconfig:"prometheus_url" required:"true"`
	Mqtt          Mqtt          `mapstructure:"mqtt" envconfig:"mqtt"`
	Metrics       []Metric      `mapstructure:"metrics" envconfig:"metrics" default:"disks_flushes:node_disk_flush_requests_total{device='sda'}"`
	Interval      time.Duration `mapstructure:"interval" envconfig:"interval" default:"15s"`
	ScrapeTimeout time.Duration `mapstructure:"scrape_timeout" envconfig:"scrape_timeout" default:"3s"`
}

type Mqtt struct {
	User               string        `mapstructure:"user" envconfig:"user"`
	Password           string        `mapstructure:"password" envconfig:"password"`
	UserFile           string        `mapstructure:"user_file" envconfig:"user_file"`
	PasswordFile       string        `mapstructure:"password_file" envconfig:"password_file"`
	Servers            []string      `mapstructure:"servers" envconfig:"servers" required:"true"`
	InsecureSkipVerify bool          `mapstructure:"insecure_skip_verify" envconfig:"insecure_skip_verify" default:"false"`
	PublishTopicPrefix string        `mapstructure:"publish_topic_prefix" envconfig:"publish_topic_prefix" default:"p2m"`
	ClientID           string        `mapstructure:"client_id" envconfig:"client_id" default:"Prometheus2MQTT"`
	RetainMessages     bool          `mapstructure:"retain_messages" envconfig:"retain_messages" default:"true"`
	PublishTimeout     time.Duration `mapstructure:"publish_timeout" envconfig:"publish_timeout" default:"5s"`
	Qos                byte          `mapstructure:"qos" envconfig:"qos" default:"1"`
	// HAPublisher defines if MQTT publishing format should be compatible with HomeAssistant
	HAPublisher     bool   `mapstructure:"ha_publisher" envconfig:"ha_publisher" default:"true"`
	DiscoveryPrefix string `mapstructure:"discovery_prefix" envconfig:"discovery_prefix" default:"homeassistant"`
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

func Load(file string) (Config, error) {
	c := Config{}

	viper.SetConfigFile(file)
	viper.SetEnvPrefix("P2M")
	viper.AutomaticEnv()

	viper.SetDefault("interval", time.Second*15)
	viper.SetDefault("scrape_timeout", time.Second*3)
	viper.SetDefault("mqtt.public_topic_prefix", "p2m")
	viper.SetDefault("mqtt.client_id", "Prometheus2MQTT")
	viper.SetDefault("mqtt.retain_messages", true)
	viper.SetDefault("mqtt.publish_timeout", time.Second*5)
	viper.SetDefault("mqtt.qos", 1)
	viper.SetDefault("mqtt.ha_publisher", true)
	viper.SetDefault("mqtt.discovery_prefix", "homeassistant")

	err := viper.ReadInConfig()
	if err != nil {
		return c, err
	}

	err = viper.Unmarshal(&c)

	return c, err
}
