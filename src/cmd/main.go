package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"
	"github.com/zhangyy8lab/tusimaServerMonitor/client"
)

// var activeServer []string
var monitorServerList []string
var Cfg *viper.Viper

type metrics struct {
	serverTotal prometheus.Gauge
	hdFailures  *prometheus.CounterVec
}

func init() {
	Cfg = viper.New()

	//Cfg.AddConfigPath("/home/ubuntu/zyy/tusimaMonitorServer/src/config/")
	Cfg.AddConfigPath("/app/config/")
	Cfg.SetConfigName("service")
	Cfg.SetConfigType("yaml")

	if err := Cfg.ReadInConfig(); err != nil {
		fmt.Println("read config file error")
		panic(err)
	}
	monitorServerList = viper.GetStringSlice("monitorServerList")
	for _, name := range Cfg.GetStringSlice("monitorServer.server") {
		monitorServerList = append(monitorServerList, strings.ReplaceAll(name, "-", "_"))
	}
}

// NewMetrics creates a new metrics instance.
func NewMetrics(reg prometheus.Registerer) *metrics {
	m := &metrics{
		serverTotal: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "alive_server_count",
			Help: "check alive server",
		}),
		hdFailures: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "custom_check_alive_server",
				Help: "custom_check_alive_server",
			},
			monitorServerList,
		),
	}

	reg.MustRegister(m.serverTotal)
	reg.MustRegister(m.hdFailures)
	return m
}

// periodicallyUpdateMetrics
func periodicallyUpdateMetrics(labels prometheus.Labels, m *metrics) {
	ticker := time.NewTicker(time.Second * 5)

	if len(monitorServerList) > 0 {
		for range ticker.C {
			lines := client.DockerPS()

			// int to float
			//activeServerStr := strconv.Itoa(len(lines) - 1)
			//activeServerFloat, _ := strconv.ParseFloat(activeServerStr, 64)
			//m.serverTotal.Set(activeServerFloat)
			if len(lines) > 0 {
				for _, name := range monitorServerList {
					labels[name] = client.CheckServetActive(name, lines)
				}
			}

			m.hdFailures.With(labels)
		}
		return
	}
}

func main() {

	// Create a non-global registry.
	reg := prometheus.NewRegistry()

	// Create new metrics and register them using the custom registry.
	m := NewMetrics(reg)

	for {
		if len(monitorServerList) > 0 {
			fmt.Println("monitorServerList:", monitorServerList)
			break
		} else {
			fmt.Println("read config failed. time.sleep 1 second")
			time.Sleep(time.Second)
			continue
		}
	}

	labels := make(prometheus.Labels)

	go periodicallyUpdateMetrics(labels, m)

	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", Cfg.Get("server.port")), nil))
}
