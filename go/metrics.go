package main

import (
	"encoding/csv"
	"os"
	"strconv"
	"time"
	"vpr/pkg/rec"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

var (
	ns        = "vpr"
	recCPUReq = prometheus.NewDesc(
		prometheus.BuildFQName(ns, "", "recommendation_requests_cpu_cores"),
		"VPR recommendation for CPU request in MilliCores",
		[]string{"namespace", "kind", "pod", "container", "alias"}, nil,
	)
	recMemReq = prometheus.NewDesc(
		prometheus.BuildFQName(ns, "", "recommendation_requests_memory_bytes"),
		"VPR recommendation for Mem request in MiB",
		[]string{"namespace", "kind", "pod", "container", "alias"}, nil,
	)
	recMemLimit = prometheus.NewDesc(
		prometheus.BuildFQName(ns, "", "recommendation_limits_memory_bytes"),
		"VPR recommendation for Mem limit in MiB",
		[]string{"namespace", "kind", "pod", "container", "alias"}, nil,
	)
	recGainCPUReq = prometheus.NewDesc(
		prometheus.BuildFQName(ns, "", "gain_requests_cpu_cores"),
		"VPR gain for CPU request in MilliCores",
		[]string{"namespace", "kind", "pod", "container", "alias"}, nil,
	)
	recGainMemReq = prometheus.NewDesc(
		prometheus.BuildFQName(ns, "", "gain_requests_memory_bytes"),
		"VPR gain for Mem request in MiB",
		[]string{"namespace", "kind", "pod", "container", "alias"}, nil,
	)
	recYoungMaxAfterGC = prometheus.NewDesc(
		prometheus.BuildFQName(ns, "", "young_max_after_gc_memory_bytes"),
		"VPR JVM max after full GC for Young Gen in MiB",
		[]string{"namespace", "kind", "pod", "container", "alias"}, nil,
	)
	recStaticMem = prometheus.NewDesc(
		prometheus.BuildFQName(ns, "", "static_jvm_memory_bytes"),
		"VPR JVM static memory baseline Mem in MiB",
		[]string{"namespace", "kind", "pod", "container", "alias"}, nil,
	)
	recOldMaxAfterFullGC = prometheus.NewDesc(
		prometheus.BuildFQName(ns, "", "old_max_after_full_gc_memory_bytes"),
		"VPR JVM max after full GC for Old Gen in MiB",
		[]string{"namespace", "kind", "pod", "container", "alias"}, nil,
	)
	recReplicas = prometheus.NewDesc(
		prometheus.BuildFQName(ns, "", "status_replicas"),
		"VPR number of replicas being a sts/dep/daemonset",
		[]string{"namespace", "kind", "pod", "container", "alias"}, nil,
	)
)

type metrics struct {
	Namespace              string
	Kind                   string
	PodGroupName           string
	ContainerName          string
	LimitAlias             string
	Replicas               int
	NewCPUReqM             float64
	NewMemReqMB            float64
	NewMemLimitMB          float64
	GainCPUReqM            float64
	GainMemReqMB           float64
	JVMYoungMaxAfterGCMB   float64
	JVMStaticMemMB         float64
	JVMOldMaxAfterFullGCMB float64
}

func init() {
	//Registering Exporter
	exporter := NewExporter()
	prometheus.MustRegister(exporter)
}

// Exporter is the struct
type Exporter struct {
}

// NewExporter is the exporter
func NewExporter() *Exporter {
	return &Exporter{}
}

// Describe are the metrics exposed
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- recCPUReq
	ch <- recMemReq
	ch <- recMemLimit
	ch <- recGainCPUReq
	ch <- recGainMemReq
	ch <- recYoungMaxAfterGC
	ch <- recStaticMem
	ch <- recOldMaxAfterFullGC
	ch <- recReplicas
}

// Collect is when metrics will be collected
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	startProm := time.Now()
	log.Info("Will collect metrics")
	e.collectPromMetrics(ch)
	end := time.Now()
	log.Info("Monitoring metrics collect finished in ", end.Sub(startProm))
}

func (e *Exporter) collectPromMetrics(ch chan<- prometheus.Metric) {
	log.Info("Will collect VPR metrics")
	for _, c := range getContainerRecommendations() {
		ch <- prometheus.MustNewConstMetric(recCPUReq, prometheus.GaugeValue, c.NewCPUReqM, c.Namespace, c.Kind, c.PodGroupName, c.ContainerName, c.LimitAlias)
		ch <- prometheus.MustNewConstMetric(recMemReq, prometheus.GaugeValue, c.NewMemReqMB, c.Namespace, c.Kind, c.PodGroupName, c.ContainerName, c.LimitAlias)
		ch <- prometheus.MustNewConstMetric(recMemLimit, prometheus.GaugeValue, c.NewMemLimitMB, c.Namespace, c.Kind, c.PodGroupName, c.ContainerName, c.LimitAlias)
		ch <- prometheus.MustNewConstMetric(recGainCPUReq, prometheus.GaugeValue, c.GainCPUReqM, c.Namespace, c.Kind, c.PodGroupName, c.ContainerName, c.LimitAlias)
		ch <- prometheus.MustNewConstMetric(recGainMemReq, prometheus.GaugeValue, c.GainMemReqMB, c.Namespace, c.Kind, c.PodGroupName, c.ContainerName, c.LimitAlias)
		ch <- prometheus.MustNewConstMetric(recYoungMaxAfterGC, prometheus.GaugeValue, c.JVMYoungMaxAfterGCMB, c.Namespace, c.Kind, c.PodGroupName, c.ContainerName, c.LimitAlias)
		ch <- prometheus.MustNewConstMetric(recStaticMem, prometheus.GaugeValue, c.JVMStaticMemMB, c.Namespace, c.Kind, c.PodGroupName, c.ContainerName, c.LimitAlias)
		ch <- prometheus.MustNewConstMetric(recOldMaxAfterFullGC, prometheus.GaugeValue, c.JVMOldMaxAfterFullGCMB, c.Namespace, c.Kind, c.PodGroupName, c.ContainerName, c.LimitAlias)
		ch <- prometheus.MustNewConstMetric(recReplicas, prometheus.GaugeValue, float64(c.Replicas), c.Namespace, c.Kind, c.PodGroupName, c.ContainerName, c.LimitAlias)
	}
}

func getContainerRecommendations() []metrics {
	f, err := os.Open(rec.OutPathCsvRecommendations)
	if err != nil {
		log.Error("Error opening csv file ", err)
		return []metrics{}
	}
	defer f.Close()

	csvReader := csv.NewReader(f)
	data, err := csvReader.ReadAll()
	if err != nil {
		log.Error("Error reading csv file ", err)
		return []metrics{}
	}
	return createMetricsList(data)
}

func createMetricsList(data [][]string) []metrics {
	var container []metrics
	for i, line := range data {
		if i > 0 && i < len(data) { // omit header line
			var rec metrics
			for j, field := range line {
				if j == 0 {
					rec.Namespace = field
				} else if j == 1 {
					rec.Kind = field
				} else if j == 2 {
					rec.PodGroupName = field
				} else if j == 3 {
					rec.Replicas, _ = strconv.Atoi(field)
				} else if j == 4 {
					rec.ContainerName = field
				} else if j == 5 {
					rec.LimitAlias = field
				} else if j == 10 {
					tmp, _ := strconv.ParseFloat(field, 64)
					rec.NewCPUReqM = tmp / 1000.0
				} else if j == 11 {
					tmp, _ := strconv.ParseFloat(field, 64)
					rec.NewMemReqMB = tmp * 1048576.0
				} else if j == 12 {
					tmp, _ := strconv.ParseFloat(field, 64)
					rec.NewMemLimitMB = tmp * 1048576.0
				} else if j == 13 {
					tmp, _ := strconv.ParseFloat(field, 64)
					rec.GainCPUReqM = tmp / 1000.0
				} else if j == 14 {
					tmp, _ := strconv.ParseFloat(field, 64)
					rec.GainMemReqMB = tmp * 1048576.0
				} else if j == 25 {
					tmp, _ := strconv.ParseFloat(field, 64)
					rec.JVMYoungMaxAfterGCMB = tmp * 1048576.0
				} else if j == 27 {
					tmp, _ := strconv.ParseFloat(field, 64)
					rec.JVMStaticMemMB = tmp * 1048576.0
				} else if j == 28 {
					tmp, _ := strconv.ParseFloat(field, 64)
					rec.JVMOldMaxAfterFullGCMB = tmp * 1048576.0
				}
			}
			container = append(container, rec)
		}
	}
	return container
}
