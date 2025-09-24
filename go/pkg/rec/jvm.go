package rec

import (
	"math"
	"strings"
	"time"
	"vpr/pkg/utils"

	"github.com/montanaflynn/stats"
	"github.com/prometheus/common/model"
	log "github.com/sirupsen/logrus"
)

const (
	//range
	queryYoungGenSize       = `sum by(pod,container)(jvm_memory_pool_committed_bytes{pod=~"$podgroup-$suffix",pool=~"PS Eden Space|G1 Eden Space|PS Survivor Space|G1 Survivor Space|Par Eden Space|Par Survivor Space|young|survivor|Eden Space|Survivor Space|ZGC Young Generation"})  / 1048576`
	queryYoungGenUsage      = `sum by(pod,container)(jvm_memory_pool_used_bytes{pod=~"$podgroup-$suffix",pool=~"PS Eden Space|G1 Eden Space|PS Survivor Space|G1 Survivor Space|Par Eden Space|Par Survivor Space|young|survivor|Eden Space|Survivor Space|ZGC Young Generation"})  / 1048576`
	queryOldGenUsage        = `sum by(pod,container)(jvm_memory_pool_used_bytes{pod=~"$podgroup-$suffix",pool=~"PS Old Gen|G1 Old Gen|CMS Old Gen|ZHeap|old|Tenured Gen|ZGC Old Generation"}) / 1048576 > 0`
	queryOldGenUsageAfterGC = `sum by(pod,container)(jvm_memory_used_after_gc_bytes{pod=~"$podgroup-$suffix",key=~"PS Old Gen|G1 Old Gen|CMS Old Gen|ZHeap|old|Tenured Gen|ZGC Old Generation"}) / 1048576 > 0` +
		` OR (sum by(pod,container)(min_over_time(jvm_memory_pool_used_bytes{pod=~"$podgroup-$suffix",pool=~"PS Old Gen|G1 Old Gen|CMS Old Gen|ZHeap|old|Tenured Gen|ZGC Old Generation"}[1h]))) / 1048576 > 0`
	//instant to calculate XMX% = (YoungPool + OldPool) / Limit
	//ZGC Young Generation is intentionally not included in the query because in Java 24 max(ZGC Old Generation)=max(ZGC Young Generation)
	queryYoungPool = `max by (container)(sum by (pod,container)(jvm_memory_pool_max_bytes{pod=~"$podgroup-$suffix",pool=~"PS Eden Space|G1 Eden Space|PS Survivor Space|G1 Survivor Space|Par Eden Space|Par Survivor Space|young|survivor|Eden Space|Survivor Space"})) / 1048576  > 0`
	queryOldPool   = `max by (container)(sum by (pod,container)(jvm_memory_pool_max_bytes{pod=~"$podgroup-$suffix",pool=~"PS Old Gen|G1 Old Gen|CMS Old Gen|ZHeap|old|Tenured Gen|ZGC Old Generation"})) / 1048576 > 0`
	//only look at the last 1 day (we dont want to look for the last 7d has it would not be fair)
	//divide by 5 because the ES query is done every 5 minutes
	queryAllocationStall = `sum by(by_app)(sum_over_time(es_query_container_java_allocation_stall_by_host_by_app_doc_count{by_host=~"$podgroup-.*"}[1d])/5)`
)

type jvmPodContainerUsage struct {
	Pod       string
	Container string
	Values    JVMStats
}

type jvmContainerUsage struct {
	Name   string
	Values JVMStats
}

// JVMContainerUsage is a struct with all JVM mem usage
type JVMContainerUsage struct {
	YoungGenSizeMB       float64
	YoungGenUsageMB      JVMStats
	OldGenUsageMB        JVMStats
	OldGenUsageAfterGcMB float64
	YoungPoolMB          float64
	OldPoolMB            float64
	AllocationStall      int
}

// JVMStats is a struct with useful stats
type JVMStats struct {
	Min        float64
	MaxAfterGC float64
	Max        float64
}

// GetPodGroupJVMUsage get jvm usage historical for a pod group
func (r *Recommender) GetPodGroupJVMUsage(namespace, podgroup, suffixKind string) map[string]JVMContainerUsage {
	result := make(map[string]JVMContainerUsage)
	nsVars := []utils.Var{{Name: "namespace", Value: namespace}, {Name: "podgroup", Value: podgroup}, {Name: "suffix", Value: suffixKind}, {Name: "interval", Value: r.Interval.String()}}

	oldGenUsageMB := r.getPodContainerJVMHistoryUsage("Old Gen", queryOldGenUsage, nsVars)
	if len(oldGenUsageMB) == 0 {
		log.Info("No Java Metrics available for pod group ", podgroup)
		return result
	}
	youngGenUsageMB := r.getPodContainerJVMHistoryUsage("Young Gen usage", queryYoungGenUsage, nsVars)
	oldGenUsageAfterGcMB := r.getPodContainerJVMHistoryUsage("Old Gen After GC", queryOldGenUsageAfterGC, nsVars)
	youngGenSizeMB := r.getPodContainerJVMHistoryUsage("Young Gen", queryYoungGenSize, nsVars)
	youngPool := r.getContainerValue(queryYoungPool, nsVars)
	OldPool := r.getContainerValue(queryOldPool, nsVars)
	//unfortunately today noway to know the container name (use pod name instead)
	allocationStall := r.getPodGroupValue(queryAllocationStall, nsVars)

	for _, elem := range oldGenUsageMB {
		result[elem.Name] = JVMContainerUsage{OldGenUsageMB: elem.Values}
	}
	for _, elem := range youngGenUsageMB {
		if val, ok := result[elem.Name]; ok {
			val.YoungGenUsageMB = elem.Values
			result[elem.Name] = val
		} else {
			result[elem.Name] = JVMContainerUsage{YoungGenUsageMB: elem.Values}
		}
	}
	for _, elem := range oldGenUsageAfterGcMB {
		if val, ok := result[elem.Name]; ok {
			val.OldGenUsageAfterGcMB = elem.Values.Max
			result[elem.Name] = val
		} else {
			result[elem.Name] = JVMContainerUsage{OldGenUsageAfterGcMB: elem.Values.Max}
		}
	}
	for _, elem := range youngGenSizeMB {
		if val, ok := result[elem.Name]; ok {
			val.YoungGenSizeMB = elem.Values.Max
			result[elem.Name] = val
		} else {
			result[elem.Name] = JVMContainerUsage{YoungGenSizeMB: elem.Values.Max}
		}
	}
	for _, elem := range youngPool {
		if val, ok := result[elem.Name]; ok {
			val.YoungPoolMB = elem.Value
			result[elem.Name] = val
		} else {
			result[elem.Name] = JVMContainerUsage{YoungPoolMB: elem.Value}
		}
	}
	for _, elem := range OldPool {
		if val, ok := result[elem.Name]; ok {
			val.OldPoolMB = elem.Value
			result[elem.Name] = val
		} else {
			result[elem.Name] = JVMContainerUsage{OldPoolMB: elem.Value}
		}
	}
	for _, elem := range allocationStall {
		//we dont have the container name, so we use the pod group name instead
		//we will only insert the value if the container which is matching the pod group already exists
		for container, val := range result {
			if strings.Contains(elem.Name, container) {
				val.AllocationStall = int(elem.Value)
				result[container] = val
			}
		}
	}

	return result
}

func (r *Recommender) getPodContainerJVMHistoryUsage(name, query string, vars []utils.Var) []jvmContainerUsage {
	result := []jvmContainerUsage{}
	resultByPod := []jvmPodContainerUsage{}
	query, err := utils.SubstVars(query, vars)
	if err != nil {
		log.Error("Error subst Vars:", err)
		return result
	}
	data, err := utils.PromQueryRange(r.PromURL, query, vars, time.Now().Add(-r.History), time.Now(), r.Interval) //instead 5*time.Minute of r.Interval which is too long (1h)
	if err != nil {
		log.Error("PromQL Range query wrong for ", query, " err ", err)
	} else {
		matrixVal, ok := data.(model.Matrix)
		if !ok {
			log.Error("Error converting to matrix for query ", query)
			return result
		}

		if len(matrixVal) == 0 {
			log.Debug("Pod has no java metrics")
			return result
		}
		for _, elem := range matrixVal {
			log.Debug("Pod : ", string(elem.Metric["pod"]))
			resultByPod = append(resultByPod, jvmPodContainerUsage{Pod: string(elem.Metric["pod"]), Container: string(elem.Metric["container"]), Values: r.GetJVMStats(elem.Values, query)})
		}
		result = append(result, getContainerSummary(resultByPod)...)
		log.Debug("Result ", name, " {Min, Max}: ", result)
	}
	return result
}

// getContainerSummary get jvm usage historical for a container
func getContainerSummary(pods []jvmPodContainerUsage) []jvmContainerUsage {
	result := []jvmContainerUsage{}
	mContainers := make(map[string]jvmContainerUsage)

	//find the max for each container
	for _, elem := range pods {
		if val, ok := mContainers[elem.Container]; ok {
			if elem.Values.Min < val.Values.Min {
				val.Values.Min = elem.Values.Min
				mContainers[elem.Container] = val
			}
			if elem.Values.Max > val.Values.Max {
				val.Values.Max = elem.Values.Max
				mContainers[elem.Container] = val
			}
			if elem.Values.MaxAfterGC > val.Values.MaxAfterGC {
				val.Values.MaxAfterGC = elem.Values.MaxAfterGC
				mContainers[elem.Container] = val
			}
		} else {
			mContainers[elem.Container] = jvmContainerUsage{Name: elem.Container, Values: JVMStats{Min: elem.Values.Min, MaxAfterGC: elem.Values.MaxAfterGC, Max: elem.Values.Max}}
		}
	}

	//convert map to slice
	for _, elem := range mContainers {
		result = append(result, elem)
	}

	return result
}

// GetJVMStats get Stats from a []model.SamplePair
func (r *Recommender) GetJVMStats(samples []model.SamplePair, query string) JVMStats {
	// Get the values
	values := make([]float64, len(samples))
	for i, sample := range samples {
		values[i] = float64(sample.Value)
	}
	// Get the stats
	max, err := stats.Max(values)
	if err != nil {
		log.Error("Error getting Max for query result ", query, " err ", err)
	}
	//only for young gen size
	if query == queryYoungGenSize {
		return JVMStats{Max: max}
	}
	// Get the min
	min := getMinExcludingZeroes(values)
	// Get the max after full GC
	maxAfterFullGC := getMaxAfterFullGC(values)
	// log.Debug("values ", values)
	jvmStats := JVMStats{Min: min, MaxAfterGC: maxAfterFullGC, Max: max}
	log.Debug("{Min, MaxAfterFullGC, Max}: ", jvmStats)

	return jvmStats
}

// get the min of an array of float64 excluding any values <= 0
func getMinExcludingZeroes(values []float64) float64 {
	min := math.MaxFloat64
	for _, value := range values {
		if value < min && value > 0 {
			min = value
		}
	}
	return min
}

func getMaxAfterFullGC(values []float64) float64 {
	maxAfterPeak := -1.0

	for i := 1; i < len(values)-1; i++ {
		prev := values[i-1]
		next := values[i+1]
		current := values[i]
		if prev > current && next > current && current > 0.0 {
			// we found a min we will keep it as a max after peak if it is greater than the previous max after peak
			if current > maxAfterPeak {
				maxAfterPeak = current
			}
		}
	}

	return maxAfterPeak
}

func (r *Recommender) getPodGroupValue(query string, vars []utils.Var) []containerValue {
	result := []containerValue{}
	query, err := utils.SubstVars(query, vars)
	if err != nil {
		log.Error("Error subst Vars:", err)
		return result
	}
	data, err := utils.PromQuery(r.PromURL, query, vars)
	if err != nil {
		log.Error("PromQL Instant query wrong for ", query, " err ", err)
	} else {
		vectorVal, ok := data.(model.Vector)
		if !ok {
			log.Error("Error converting to Vector for query ", query)
			return result
		}
		for _, elem := range vectorVal {
			result = append(result, containerValue{Name: string(elem.Metric["by_app"]), Value: float64(elem.Value)})
		}
	}
	return result
}
