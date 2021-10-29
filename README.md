# Prometheus-to-MQTT [![Docker Pulls](https://img.shields.io/docker/pulls/krzysztofgzocha/prometheus2mqtt)](https://hub.docker.com/r/krzysztofgzocha/prometheus2mqtt)
Scraping metrics from Prometheus and sending them to MQTT broker.

## Why?
I was trying to build small dashboard on my phone, which would pull the metrics from MQTT broker.
I've noticed that I already have a lot of metrics stored in prometheus, so I wanted to re-use them.

## How?
Edit config.yaml file, build the program and run it:
```bash
CGO_ENABLED=0 go build -o prometheus2mqtt . && ./prometheus2mqtt 
```

### Docker
...or use this docker image: https://hub.docker.com/r/krzysztofgzocha/prometheus2mqtt
Currently just a few OS/arch are ready and the build is not yet automated.

## Config options
```yaml
prometheus_url: http://prometheus:9090
interval: 15s
scrape_timeout: 3s
mqtt:
  user: admin
# user_file: /var/secret/user # Useful when using docker secrets 
  password: admin
# password_file: /var/secret/user # Useful when using docker secrets
  servers:
    - mqtt://mosquitto:1883
  insecure_skip_verify: true
  publish_topic_prefix: p2m
  client_id: Prometheus2MQTT
  retain_messages: true
  publish_timeout: 5s
  qos: 1
  ha_publisher: true # Should it be compatible with HomeAssistant MQTT discovery?
  discovery_prefix: homeassistant
metrics:
  - name: "Health: Prometheus"
    query: up{job='prometheus'}
```

### Metrics format
In order to configure metrics we have to specify 2 value for each of them: 
- `name`: easy to read name, which will be sent to MQTT broker and could be picked up in the topic `p2m/<name>`. 
All non-alfanumeric characters will be replaced with `_` when constructing MQTT topic.
- `query`: used to query Prometheus
