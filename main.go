package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"net/http"

	"github.com/MichaelS11/go-dht"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	version string
	commit  string
)

var (
	tempF = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "dht22_temperature_fahrenheit",
		Help: "The temperature in F",
	})
	humidityM = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "dht22_humidity",
		Help: "The humity",
	})
	gatheringDuration = promauto.NewSummary(prometheus.SummaryOpts{
		Name: "gatheringduration",
		Help: "The duration of data gatherings",
	})
	GPIO = os.Getenv("GPIO")
)

func setTemps(read *dht.DHT) {
	timer := prometheus.NewTimer(gatheringDuration)
	defer timer.ObserveDuration()
	humidity, temperature, err := read.ReadRetry(11)
	if err != nil {
		log.Println("Read error: ", err)
		return
	}
	tempF.Set(temperature)
	humidityM.Set(humidity)
}

func main() {
	log.Println("Version: " + version)
	log.Println("Commit: " + commit)
	if GPIO == "" {
		log.Println("Invalid GPIO pin")
		GPIO = "GPIO27"
	}

	err := dht.HostInit()
	if err != nil {
		log.Fatal("HostInit error: ", err)
		return
	}

	read, err := dht.NewDHT(GPIO, dht.Fahrenheit, "")
	if err != nil {
		log.Fatal("NewDHT error: ", err)
		return
	}

	stopChan := make(chan os.Signal, 5)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		setTemps(read)
		for {
			select {
			case <-stopChan:
				return
			case <-time.After(10 * time.Second):
				setTemps(read)
			}
		}
	}()
	go func() {
		log.Println("Listening on port 8001")
		http.Handle("/metrics", promhttp.Handler())
		err = http.ListenAndServe(":8001", nil)
		if err != http.ErrServerClosed {
			fmt.Println(err)
			os.Exit(1)
		}
	}()

	<-stopChan
}
