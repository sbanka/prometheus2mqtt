package main

import (
	"context"
	"crypto/tls"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/krzysztof-gzocha/prometheus2mqtt/config"
	"github.com/krzysztof-gzocha/prometheus2mqtt/prometheus"
	"github.com/krzysztof-gzocha/prometheus2mqtt/publisher"
	"github.com/krzysztof-gzocha/prometheus2mqtt/ticker"
	"github.com/prometheus/client_golang/api"
	promHttp "github.com/prometheus/client_golang/api/prometheus/v1"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

func main() {
	logger := log.New(os.Stderr, "", log.LstdFlags)
	logger.Printf("Starting Prometheus2MQTT (ver: %s)\n", config.Version)
	ctx, terminate := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt)
	defer terminate()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatalf("Could not load the configuration: %s", err.Error())
	}

	transport := defaultTransport(cfg.Interval)
	prometheusAPI := getPrometheusClient(logger, cfg.PrometheusUrl, transport)
	prometheusClient := prometheus.NewScraper(prometheusAPI, logger)
	mqttClient := mqtt.NewClient(mqttClientOptions(cfg.Mqtt, logger))
	t := mqttClient.Connect()

	select {
	case <-t.Done():
		if t.Error() != nil {
			logger.Fatalf("Couldn't connect to MQTT broker: %v", t.Error())
		}
	case <-ctx.Done():
		logger.Printf("Received signal to stop")
		os.Exit(0)
	}

	if !mqttClient.IsConnected() {
		logger.Fatalf("couldn't connect to mqtt")
	}

	var mqttPub publisher.Publisher
	mqttPub = publisher.NewSimple(cfg.Mqtt, mqttClient, logger)
	if cfg.Mqtt.HAPublisher {
		mqttPub = publisher.NewHomeAssistant(cfg.Mqtt, mqttClient, logger)
	}

	scrapingTicker := ticker.NewTicker(cfg, prometheusClient, mqttPub, logger)
	scrapingTicker.Start(ctx)
	mqttClient.Disconnect(50)
}

func mqttClientOptions(mqttConfig config.Mqtt, logger *log.Logger) *mqtt.ClientOptions {
	mqtt.CRITICAL = logger
	mqtt.ERROR = logger

	clientId := mqttConfig.ClientID
	if clientId == "" {
		randomId := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(99999)
		clientId = "p2m." + strconv.Itoa(randomId)
	}

	logger.Printf("ClientID: %s\n", clientId)
	cfg := mqtt.NewClientOptions()
	cfg.
		SetClientID(clientId).
		SetResumeSubs(true).
		SetTLSConfig(&tls.Config{InsecureSkipVerify: mqttConfig.InsecureSkipVerify}).
		SetConnectRetry(true).
		SetConnectRetryInterval(time.Second * 5).
		SetConnectTimeout(time.Second * 2).
		SetAutoReconnect(true).
		SetUsername(mqttConfig.GetUser()).
		SetPassword(mqttConfig.GetPassword())

	servers, err := mqttConfig.ServersUrls()
	if err != nil {
		logger.Fatalf("Could not parse MQTT server url due to: %s", err.Error())
	}
	cfg.Servers = servers
	cfg.OnConnectAttempt = func(url *url.URL, tlsCfg *tls.Config) *tls.Config {
		logger.Printf("Attempting to connect with MQTT broker: %s\n", url.String())
		return tlsCfg
	}
	cfg.OnConnect = func(_ mqtt.Client) {
		logger.Println("Connected with MQTT broker")
	}
	cfg.OnReconnecting = func(_ mqtt.Client, _ *mqtt.ClientOptions) {
		logger.Println("Reconnecting with MQTT broker...")
	}
	cfg.OnConnectionLost = func(_ mqtt.Client, err error) {
		logger.Printf("[ERROR] Connection to MQTT broker was lost due to: %+v\n", err)
	}

	return cfg
}

func defaultTransport(interval time.Duration) *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout: interval,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		DisableKeepAlives:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       interval,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

func getPrometheusClient(logger *log.Logger, url string, trip http.RoundTripper) v1.API {
	promClient, err := api.NewClient(api.Config{
		Address:      url,
		RoundTripper: trip,
	})
	if err != nil {
		logger.Fatal(err.Error())
	}

	return promHttp.NewAPI(promClient)
}
