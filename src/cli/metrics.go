package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"isp-checker/probes"
)

var (
	runCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "isp_checker_runs_total",
		Help: "Total runs executed by the CLI",
	})
	probeResults = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "isp_checker_probe_results_total",
		Help: "Counts of probe results by status",
	}, []string{"probe", "status"})
)

func init() {
	prometheus.MustRegister(runCounter)
	prometheus.MustRegister(probeResults)
}

func startMetricsServer(addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	return http.ListenAndServe(addr, mux)
}

func recordRunMetrics(_ RunResult) {
	runCounter.Inc()
}

func recordProbeMetrics(res probes.Result) {
	probeResults.WithLabelValues(res.Name, res.Status).Inc()
}
