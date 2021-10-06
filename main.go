package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/krzysztof-gzocha/prometheus2mqtt/prometheus"
	"github.com/prometheus/client_golang/api"
	promHttp "github.com/prometheus/client_golang/api/prometheus/v1"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

func main() {
	ctx, terminate := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt)
	defer terminate()

	var prometheusUrl, mqttUrl, metricsToScrape string
	var intervalInSeconds int

	flag.StringVar(&prometheusUrl, "prometheus", "http://localhost:9090", "Full Prometheus URL including protocol and port")
	flag.StringVar(&mqttUrl, "mqtt", "http://localhost:9090", "Full MQTT Broker URL including protocol and port")
	flag.StringVar(&metricsToScrape, "metrics", "up", "Metrics to scrape split by comma")
	flag.IntVar(&intervalInSeconds, "interval", 5, "Scraping interval in seconds")
	flag.Parse()

	log.Printf("Starting scraping for %d metric(s) every %d seconds\n", len(strings.Split(metricsToScrape, ",")), intervalInSeconds)

	transport := defaultTransport(intervalInSeconds)
	prometheusAPI, err := getPrometheusClient(prometheusUrl, transport)
	if err != nil {
		log.Fatal(err.Error())
	}
	client := prometheus.NewClient(prometheusAPI)
	ticker := time.NewTicker(time.Second * time.Duration(intervalInSeconds))

	for {
		select {
		case <-ticker.C:
			tick(ctx, client, intervalInSeconds, metricsToScrape)
		case <-ctx.Done():
			log.Printf("Gracefully stopping..")
			os.Exit(0)
		}
	}
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

func getPrometheusClient(url string, trip http.RoundTripper) (v1.API, error) {
	promClient, err := api.NewClient(api.Config{
		Address:      url,
		RoundTripper: trip,
	})
	if err != nil {
		log.Fatal(err.Error())
	}

	return promHttp.NewAPI(promClient), nil
}

func tick(ctx context.Context, client prometheus.Client, intervalInSeconds int, metricsToScrape string) {
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
	metrics, err := client.Scrape(
		ctxTimeout,
		strings.Split(metricsToScrape, ",")...,
	)
	cancel()
	if err == context.DeadlineExceeded {
		log.Printf("Scraping metrics exceeded timeout\n")
	} else if err != nil {
		log.Printf("Error when scraping for metrics: %s\n", err.Error())
	}

	for n, m := range metrics {
		// @todo send to mqtt
		log.Printf("Metric \t%s\t has value:\t%.0f\tcollected at:\t%s\n", n, m.Value, m.Time.String())
	}
}
