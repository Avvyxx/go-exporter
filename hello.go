package main

import (
  "fmt"
  "log"
  "net/http"
  "os"
  "os/user"
  "strings"

  "github.com/prometheus/client_golang/prometheus"
  "github.com/prometheus/client_golang/prometheus/promhttp"
  "github.com/prometheus/procfs"
)

type ProcessCollector struct {
  infoDesc      *prometheus.Desc
  cpuDesc       *prometheus.Desc
  memoryDesc    *prometheus.Desc
  hostnameLabel string
}

func NewProcessCollector() *ProcessCollector {
  hostname, err := os.Hostname()
  if err != nil {
    hostname = "unknown"
  }

  return &ProcessCollector{
    hostnameLabel: hostname,
    infoDesc: prometheus.NewDesc(
      "process_info",
      "Static info about running processes by user and host.",
      []string{"user", "process", "host", "pid"},
      nil,
    ),
    cpuDesc: prometheus.NewDesc(
      "process_cpu_seconds",
      "CPU usage of the process in seconds.",
      []string{"user", "process", "host", "pid"},
      nil,
    ),
    memoryDesc: prometheus.NewDesc(
      "process_memory_bytes",
      "Memory usage of the process in bytes.",
      []string{"user", "process", "host", "pid"},
      nil,
    ),
  }
}

func (collector *ProcessCollector) Describe(ch chan<- *prometheus.Desc) {
  ch <- collector.infoDesc
  ch <- collector.cpuDesc
  ch <- collector.memoryDesc
}

func (collector *ProcessCollector) Collect(ch chan<- prometheus.Metric) {
  procs, err := procfs.AllProcs()
  if err != nil {
    log.Println("Error fetching processes:", err)
    return
  }

  for _, p := range procs {
    stat, err := p.Stat()
    if err != nil {
      continue
    }

    status, err := p.NewStatus()
    if err != nil {
      continue
    }

    uid := status.UIDs[0]
    pid := fmt.Sprint(p.PID)

    name, err := p.Comm()
    if err != nil {
      continue
    }
    name = strings.TrimSpace(name)

    cpu := stat.CPUTime()

    mem := float64(stat.ResidentMemory())

    // Emit metrics
    ch <- prometheus.MustNewConstMetric(
      collector.infoDesc,
      prometheus.GaugeValue,
      1,
      uid,
      name,
      collector.hostnameLabel,
      pid,
    )
    ch <- prometheus.MustNewConstMetric(
      collector.cpuDesc,
      prometheus.CounterValue,
      cpu,
      uid,
      name,
      collector.hostnameLabel,
      pid,
    )
    ch <- prometheus.MustNewConstMetric(
      collector.memoryDesc,
      prometheus.GaugeValue,
      mem,
      uid,
      name,
      collector.hostnameLabel,
      pid,
    )
  }
}

func main() {
  collector := NewProcessCollector()
  prometheus.MustRegister(collector)

  http.Handle("/metrics", promhttp.Handler())

  log.Fatal(http.ListenAndServe(":9100", nil))
}
