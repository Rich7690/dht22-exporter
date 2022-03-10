package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MichaelS11/go-dht"

	"github.com/creativeprojects/go-selfupdate"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	version       string = "0.0.0"
	commit        string
	DisableUpdate bool = os.Getenv("DISABLE_UPDATE") == "true"
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

func doSelfUpdate() {
	selfupdate.SetLogger(log.Default()) // enable when debug logging is needed
	updater, err := selfupdate.NewUpdater(selfupdate.Config{Validator: &selfupdate.ChecksumValidator{UniqueFilename: "checksums.txt"}})
	if err != nil {
		log.Printf("Error creating updater: %v\n", err)
		return
	}
	if DisableUpdate {
		latest, found, err := updater.DetectLatest("rtdev7690/dht22-exporter")
		if err != nil {
			log.Printf("Error finding latest version: %v\n", err)
			return
		}
		if found {
			log.Println("Found latest version: ", latest)
		} else {
			log.Println("Couldn't find latest version")
		}

		return
	}

	latest, err := updater.UpdateSelf(version, "rtdev7690/dht22-exporter")
	if err != nil {
		log.Println("Binary update failed:", err)
		return
	}
	log.Println("Latest version: ", latest.Version())
	if latest.Equal(version) {
		// latest version is the same as current version. It means current binary is up to date.
		log.Println("Current binary is the latest version", version)
	} else {
		log.Println("Successfully updated to version", latest.Version())
		log.Println("Release note:\n", latest.ReleaseNotes)
		log.Println("Exiting.")
		os.Exit(0)
	}
}

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
	doSelfUpdate()
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
		http.Handle("/health", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
		err = http.ListenAndServe(":8001", nil)
		if err != http.ErrServerClosed {
			fmt.Println(err)
			os.Exit(1)
		}
	}()

	<-stopChan
}
