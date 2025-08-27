package main

import (
  "fmt"
  "log"
  "net/http"
  "os"
  "strings"
  "flag"

  "github.com/prometheus/client_golang/prometheus"
  "github.com/prometheus/client_golang/prometheus/promhttp"
  "github.com/prometheus/procfs"
)

type ProcessCollector struct {
  memoryDesc    *prometheus.Desc
  systemTime    *prometheus.Desc
  userTime      *prometheus.Desc
  hostnameLabel string
}

func NewProcessCollector(hostname string) *ProcessCollector {
  return &ProcessCollector{
    hostnameLabel: hostname,
    memoryDesc: prometheus.NewDesc(
      "process_resident_memory_bytes",
      "Memory usage of the process in bytes.",
      []string{"uid", "process", "host", "pid"},
      nil,
    ),
    systemTime: prometheus.NewDesc(
      "process_system_seconds_total",
      "Amount of time spent in system mode.",
      []string{"uid", "process", "host", "pid"},
      nil,
    ),
    userTime: prometheus.NewDesc(
     "process_user_seconds_total",
     "Amount of time spent in user mode.",
     []string{"uid", "process", "host", "pid"},
     nil,
    ),
  }
}

func (collector *ProcessCollector) Describe(ch chan<- *prometheus.Desc) {
  ch <- collector.memoryDesc
  ch <- collector.systemTime
  ch <- collector.userTime
}

func (collector *ProcessCollector) Collect(ch chan<- prometheus.Metric) {
  fs, err    := procfs.NewFS("/proc")

  if err != nil {
    log.Println("Error creating FS: ", err)
    return
  }

  procs, err := fs.AllProcs()

  if err != nil {
    log.Println("Error fetching processes: ", err)
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

    name, err := p.Comm()
    if err != nil {
      continue
    }

    clk_tck   := 100

    user_secs := float64(stat.UTime) / float64(clk_tck)
    sys_secs  := float64(stat.STime) / float64(clk_tck)

    uid := fmt.Sprint(status.UIDs[0])
    pid := fmt.Sprint(p.PID)
    proc_name := strings.TrimSpace(name)
    mem_used_bytes := float64(stat.ResidentMemory())

    // Emit metrics
    ch <- prometheus.MustNewConstMetric(
      collector.memoryDesc,
      prometheus.GaugeValue,
      mem_used_bytes,
      uid,
      proc_name,
      collector.hostnameLabel,
      pid,
    )
    ch <- prometheus.MustNewConstMetric(
      collector.systemTime,
      prometheus.CounterValue,
      sys_secs,
      uid,
      proc_name,
      collector.hostnameLabel,
      pid,
    )
    ch <- prometheus.MustNewConstMetric(
      collector.userTime,
      prometheus.CounterValue,
      user_secs,
      uid,
      proc_name,
      collector.hostnameLabel,
      pid,
    )
  }
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
