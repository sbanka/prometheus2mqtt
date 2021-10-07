package main

import (
	"context"
	"crypto/tls"
	"flag"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/krzysztof-gzocha/prometheus2mqtt/prometheus"
	"github.com/prometheus/client_golang/api"
	promHttp "github.com/prometheus/client_golang/api/prometheus/v1"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

func main() {
	logger := log.New(os.Stderr, "", log.LstdFlags)
	ctx, terminate := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt)
	defer terminate()

	var prometheusUrl, mqttUrl, metricsToScrape string
	var intervalInSeconds int

	flag.StringVar(&prometheusUrl, "prometheus", "http://localhost:9090", "Full Prometheus URL including protocol and port")
	flag.StringVar(&mqttUrl, "mqtt", "mqtt://localhost:9090", "Full MQTT Broker URL including protocol and port")
	flag.StringVar(&metricsToScrape, "metrics", "up", "Metrics to scrape split by comma")
	flag.IntVar(&intervalInSeconds, "interval", 5, "Scraping interval in seconds")
	flag.Parse()

	transport := defaultTransport(intervalInSeconds)
	prometheusAPI, err := getPrometheusClient(logger, prometheusUrl, transport)
	if err != nil {
		logger.Fatal(err.Error())
	}
	prometheusClient := prometheus.NewClient(prometheusAPI)
	ticker := time.NewTicker(time.Second * time.Duration(intervalInSeconds))

	mqttClient := mqtt.NewClient(mqttClientOptions(mqttUrl, logger))
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
	} else {
		logger.Println("Connected to MQTT broker")
	}

	logger.Printf(
		"Starting scraping for %d metric(s) every %d seconds\n",
		len(strings.Split(metricsToScrape, ",")),
		intervalInSeconds,
	)

	for {
		select {
		case <-ticker.C:
			tick(ctx, logger, prometheusClient, mqttClient, intervalInSeconds, metricsToScrape)
		case <-ctx.Done():
			logger.Printf("Received signal to stop")
			mqttClient.Disconnect(50)
			os.Exit(0)
		}
	}
}

func mqttClientOptions(mqttUrl string, logger *log.Logger) *mqtt.ClientOptions {
	randomId := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(9999)

	mqttServers := make([]*url.URL, 0)
	serverUrl, err := url.Parse(mqttUrl)
	if err != nil {
		logger.Fatal(err.Error())
	}
	clientId := "prometheus2mqtt." + strconv.Itoa(randomId)
	logger.Printf("ClientID: %s\n", clientId)
	config := mqtt.NewClientOptions()
	config.
		SetClientID(clientId).
		SetResumeSubs(true).
		SetTLSConfig(&tls.Config{InsecureSkipVerify: true}).
		SetConnectRetry(true).
		SetConnectRetryInterval(time.Second * 5).
		SetConnectTimeout(time.Second * 2)

	config.Servers = append(mqttServers, serverUrl)
	config.OnConnectAttempt = func(url *url.URL, tlsCfg *tls.Config) *tls.Config {
		logger.Printf("Attempting to connect with MQTT broker: %s\n", url.String())
		return tlsCfg
	}
	config.OnConnect = func(_ mqtt.Client) {
		logger.Println("Connected with MQTT broker")
	}
	config.OnReconnecting = func(_ mqtt.Client, _ *mqtt.ClientOptions) {
		logger.Println("Reconnecting with MQTT broker...")
	}

	return config
}

func defaultTransport(intervalInSeconds int) *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout: time.Duration(intervalInSeconds) * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		DisableKeepAlives:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       time.Duration(intervalInSeconds) * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

func getPrometheusClient(logger *log.Logger, url string, trip http.RoundTripper) (v1.API, error) {
	promClient, err := api.NewClient(api.Config{
		Address:      url,
		RoundTripper: trip,
	})
	if err != nil {
		logger.Fatal(err.Error())
	}

	return promHttp.NewAPI(promClient), nil
}

func tick(
	ctx context.Context,
	logger *log.Logger,
	prometheusClient prometheus.Client,
	mqttClient mqtt.Client,
	intervalInSeconds int,
	metricsToScrape string,
) {
	defer func() {
		e := recover()
		if e != nil {
			log.Printf("Recovering from panic: %+v\n", e)
		}
	}()

	ctxTimeout, cancel := context.WithTimeout(
		ctx,
		(time.Duration(intervalInSeconds)*time.Second)-(100*time.Millisecond),
	)

	metrics, err := prometheusClient.Scrape(
		ctxTimeout,
		strings.Split(metricsToScrape, ",")...,
	)
	cancel()
	if err == context.DeadlineExceeded {
		logger.Printf("Scraping metrics exceeded timeout\n")
	} else if err != nil {
		logger.Printf("Error when scraping for metrics: %s\n", err.Error())
	}

	for n, m := range metrics {
		t := mqttClient.Publish("prometheus2mqtt/"+n, 0, true, m.Value)
		if !t.WaitTimeout(time.Millisecond * 250) {
			logger.Printf("Publishing metric %s took over 250ms, continuing...\n", n)
			continue
		}

		logger.Printf("Metric %s (value: %.2f) was send to MQTT\n", n)
	}
}
