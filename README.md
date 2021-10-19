# Prometheus-to-MQTT
Scraping metrics from Prometheus and sending them to MQTT broker.

## Why?
I was trying to build small dashboard on my phone, which would pull the metrics from MQTT broker.
I've noticed that I already have a lot of metrics stored in prometheus, so I wanted to re-use them.

## How?
I'm using [`kelseyhightower/envconfig`](https://github.com/kelseyhightower/envconfig), so you can configure all the parameters via environment variables and run it via:
```
go run main.go
```
or build it and run:
```
go build . && ./prometheus2mqtt
```

## Config options
```go
type Config struct {
	PrometheusUrl string            `envconfig:"prometheus_url" required:"true"`
	MqttBroker    Mqtt              `envconfig:"mqtt"`
	Metrics       map[string]string `envconfig:"metrics" default:"up_prometheus:up{job='prometheus'}"`
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
	ClientID           string        `envconfig:"client_id"`
	RetainMessages     bool          `envconfig:"retain_messages" default:"true"`
	PublishTimeout     time.Duration `envconfig:"publish_timeout" default:"5s"`
	Qos                byte          `envconfig:"qos" default:"1"`
}
```
To configure MQTT user for example use `P2M_MQTT_USER` environment variable.

### Metrics format
In order to configure metrics we have to specify 2 value for each of them: 
- `alias`: easy to read name, which will be sent to MQTT broker and could be picked up in the topic `p2m/<alias>`
- `query`: used to query Prometheus

Format of metrics flag is `alias1:query1,alias2:query2`.
