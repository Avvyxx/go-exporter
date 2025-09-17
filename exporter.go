package main

import (
  "fmt"
  "log"
  "net/http"
  "os"
  "flag"

  "github.com/prometheus/client_golang/prometheus"
  "github.com/prometheus/client_golang/prometheus/promhttp"
  "github.com/prometheus/procfs"
)

type ProcessCollector struct {
  totalMemoryDesc     *prometheus.Desc
  availableMemoryDesc *prometheus.Desc
  hostnameLabel string
}

func NewProcessCollector(hostname string) *ProcessCollector {
  return &ProcessCollector{
    hostnameLabel: hostname,
		totalMemoryDesc: prometheus.NewDesc(
			"total_memory_bytes",
			"Total memory in bytes.",
			[]string{"host"},
			nil,
		),
    availableMemoryDesc: prometheus.NewDesc(
      "current_memory_available_bytes",
      "Free memory in bytes.",
      []string{"host"},
      nil,
    ),
  }
}

func (collector *ProcessCollector) Describe(ch chan<- *prometheus.Desc) {
  ch <- collector.totalMemoryDesc
  ch <- collector.availableMemoryDesc
}

func (collector *ProcessCollector) Collect(ch chan<- prometheus.Metric) {
  fs, err := procfs.NewFS("/proc")

  if err != nil {
    log.Println("Error creating FS: ", err)
    return
  }

  meminfo, err := fs.Meminfo()

  if err != nil {
    log.Println("Error fetching memory info: ", err)
    return
  }

  // Emit metrics
  ch <- prometheus.MustNewConstMetric(
    collector.totalMemoryDesc,
    prometheus.GaugeValue,
    float64(*meminfo.MemTotal),
    collector.hostnameLabel,
  )
  ch <- prometheus.MustNewConstMetric(
    collector.availableMemoryDesc,
    prometheus.GaugeValue,
    float64(*meminfo.MemAvailable),
    collector.hostnameLabel,
  )
}

func main() {
  name, err := os.Hostname()

  if err != nil {
    panic(err)
  }

  hostname := flag.String("hostname", name, "hostname to pass to exported metrics")
  port := flag.String("port", "9100", "port to bind program")

  flag.Parse()

  // create custom registry to only export what we want to export
  registry := prometheus.NewRegistry()

  // register our custom exports
  collector := NewProcessCollector(*hostname)
  registry.MustRegister(collector)

  handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
  http.Handle("/metrics", handler)

  fmt.Println("Exporting on port", *port)
  fmt.Println("Metrics will have", "\"" + *hostname + "\"", "set as the hostname")

  log.Fatal(http.ListenAndServe(":" + *port, nil))
}
