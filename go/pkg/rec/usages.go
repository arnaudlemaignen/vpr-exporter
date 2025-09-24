package rec

import (
	"time"
	"vpr/pkg/utils"

	"github.com/montanaflynn/stats"
	"github.com/prometheus/common/model"
	log "github.com/sirupsen/logrus"
)

const (
	queryCPUUsage = `max by(container)(rate (container_cpu_usage_seconds_total{namespace=~"$namespace",pod=~"$podgroup$suffix",container!="",container!="POD"}[$interval])) * 1000`
	queryMemUsage = `max by(container)(container_memory_working_set_bytes{namespace=~"$namespace",pod=~"$podgroup$suffix",container!="",container!="POD"}) / 1048576`
)

type containerUsage struct {
	Name   string
	Values Stats
}

// ContainerUsage is a struct with all containers CPU/mem usage
type ContainerUsage struct {
	CPUUsageM  Stats
	MemUsageMB Stats
}

// Stats is a struct with useful stats
type Stats struct {
	Min        float64
	Mean       float64
	Percentile float64
	Max        float64
}

// GetPodGroupUsage get cpu/mem usage historical for a pod group
func (r *Recommender) GetPodGroupUsage(namespace, podgroup, suffixKind string) map[string]ContainerUsage {
	result := make(map[string]ContainerUsage)
	nsVars := []utils.Var{{Name: "namespace", Value: namespace}, {Name: "podgroup", Value: podgroup}, {Name: "suffix", Value: suffixKind}, {Name: "interval", Value: r.Interval.String()}}

	cpuUsage := r.getContainerUsage(queryCPUUsage, nsVars)
	memUsage := r.getContainerUsage(queryMemUsage, nsVars)

	for _, elem := range cpuUsage {
		result[elem.Name] = ContainerUsage{CPUUsageM: elem.Values}
	}
	for _, elem := range memUsage {
		if val, ok := result[elem.Name]; ok {
			val.MemUsageMB = elem.Values
			result[elem.Name] = val
		} else {
			result[elem.Name] = ContainerUsage{MemUsageMB: elem.Values}
		}
	}

	return result
}

func (r *Recommender) getContainerUsage(query string, vars []utils.Var) []containerUsage {
	result := []containerUsage{}
	query, err := utils.SubstVars(query, vars)
	if err != nil {
		log.Error("Error subst Vars:", err)
		return result
	}
	data, err := utils.PromQueryRange(r.PromURL, query, vars, time.Now().Add(-r.History), time.Now(), r.Interval)
	if err != nil {
		log.Error("PromQL Range query wrong for ", query, " err ", err)
	} else {
		matrixVal, ok := data.(model.Matrix)
		if !ok {
			log.Error("Error converting to matrix for query ", query)
			return result
		}
		for _, elem := range matrixVal {
			result = append(result, containerUsage{Name: string(elem.Metric["container"]), Values: r.GetStats(elem.Values, query)})
		}
	}
	return result
}

// GetStats get Stats from a []model.SamplePair
func (r *Recommender) GetStats(samples []model.SamplePair, query string) Stats {
	// Get the values
	values := make([]float64, len(samples))
	for i, sample := range samples {
		values[i] = float64(sample.Value)
	}
	// Get the stats
	min, err := stats.Min(values)
	if err != nil {
		log.Error("Error getting Min for query result ", query, " err ", err)
	}
	mean, err := stats.Mean(values)
	if err != nil {
		log.Error("Error getting Mean for query result ", query, " err ", err)
	}
	percent := 95.0
	if query == queryCPUUsage {
		percent = r.TargetCPUPercentile
	} else if query == queryMemUsage {
		percent = r.TargetMemPercentile
	}
	percentile, err := stats.Percentile(values, percent)
	if err != nil {
		log.Error("Error getting Percentile for query result ", query, " err ", err)
	}
	max, err := stats.Max(values)
	if err != nil {
		log.Error("Error getting Max for query result ", query, " err ", err)
	}

	return Stats{Min: min, Mean: mean, Percentile: percentile, Max: max}
}
