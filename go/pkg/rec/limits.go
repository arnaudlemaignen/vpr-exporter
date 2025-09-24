package rec

import (
	"vpr/pkg/utils"

	"github.com/prometheus/common/model"
	log "github.com/sirupsen/logrus"
)

const (
	queryCPULimit = `max by (container)(kube_pod_container_resource_limits_cpu_cores{namespace=~"$namespace",pod=~"$podgroup$suffix"}) * 1000`
	queryMemLimit = `max by (container)(kube_pod_container_resource_limits_memory_bytes{namespace=~"$namespace",pod=~"$podgroup$suffix"}) / 1048576`
	queryCPUReq   = `max by (container)(kube_pod_container_resource_requests_cpu_cores{namespace=~"$namespace",pod=~"$podgroup$suffix"}) * 1000`
	queryMemReq   = `max by (container)(kube_pod_container_resource_requests_memory_bytes{namespace=~"$namespace",pod=~"$podgroup$suffix"}) / 1048576`
)

// ContainerLimits is a struct with all containers limits and requests
type ContainerLimits struct {
	CPUReqM    float64
	MemReqMB   float64
	CPULimitM  float64
	MemLimitMB float64
}

type containerValue struct {
	Name  string
	Value float64
}

// GetPodGroupLimits get Pod groups limits
func (r *Recommender) GetPodGroupLimits(namespace, podgroup, suffixKind string) map[string]ContainerLimits {
	result := make(map[string]ContainerLimits)
	nsVars := []utils.Var{{Name: "namespace", Value: namespace}, {Name: "podgroup", Value: podgroup}, {Name: "suffix", Value: suffixKind}}

	cpuReq := r.getContainerValue(queryCPUReq, nsVars)
	memReq := r.getContainerValue(queryMemReq, nsVars)
	cpuLimit := r.getContainerValue(queryCPULimit, nsVars)
	memLimit := r.getContainerValue(queryMemLimit, nsVars)

	for _, elem := range cpuReq {
		result[elem.Name] = ContainerLimits{CPUReqM: elem.Value}
	}
	for _, elem := range memReq {
		if val, ok := result[elem.Name]; ok {
			val.MemReqMB = elem.Value
			result[elem.Name] = val
		} else {
			result[elem.Name] = ContainerLimits{MemReqMB: elem.Value}
		}
	}
	for _, elem := range cpuLimit {
		if val, ok := result[elem.Name]; ok {
			val.CPULimitM = elem.Value
			result[elem.Name] = val
		} else {
			result[elem.Name] = ContainerLimits{CPULimitM: elem.Value}
		}
	}
	for _, elem := range memLimit {
		if val, ok := result[elem.Name]; ok {
			val.MemLimitMB = elem.Value
			result[elem.Name] = val
		} else {
			result[elem.Name] = ContainerLimits{MemLimitMB: elem.Value}
		}
	}
	return result
}

func (r *Recommender) getContainerValue(query string, vars []utils.Var) []containerValue {
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
			result = append(result, containerValue{Name: string(elem.Metric["container"]), Value: float64(elem.Value)})
		}

	}
	return result
}
