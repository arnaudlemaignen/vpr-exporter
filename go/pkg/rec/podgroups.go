package rec

import (
	"vpr/pkg/utils"

	"github.com/prometheus/common/model"
	log "github.com/sirupsen/logrus"
)

const (
	sts            = "statefulset"
	rs             = "replicaset"
	ds             = "daemonset"
	dep            = "deployment"
	cron           = "cronjob"
	sparkDrivers   = "spark_driver"
	sparkExecutors = "spark_executor"
	//all kinds which have at least one pod
	querySts = `max by(statefulset,namespace)(kube_statefulset_status_replicas_ready{namespace=~"$namespace"}) > 0`
	// queryRs  = `max by(replicaset,namespace)(kube_replicaset_status_replicas{namespace=~"$namespace"}) > 0`
	queryDs       = `max by(daemonset,namespace)(kube_daemonset_status_number_ready{namespace=~"$namespace"}) > 0`
	queryDep      = `max by(deployment,namespace)(kube_deployment_status_replicas_ready{namespace=~"$namespace"}) > 0`
	queryCron     = `max by(cronjob,namespace)(kube_cronjob_status_active{namespace=~"$namespace"})`
	queryDriver   = `max by (spark_driver,namespace)(label_replace(kube_pod_container_info{namespace=~"$namespace",container=~"spark-kubernetes-driver"}, "spark_driver", "$1", "pod", "(.*)"))`
	queryExecutor = `max by (spark_executor,namespace)(label_replace(kube_pod_container_info{namespace=~"$namespace",container=~"spark-kubernetes-executor"}, "spark_executor", "$1", "pod", "(.*)-\\w+-exec-\\d+"))`
)

// PodGroup is a struct of PodGroup
type PodGroup struct {
	Kind      string
	Name      string
	Namespace string
	Suffix    string
	Count     int
}

// GetPodGroups get Pod groups sts/dep/ds/rs
func (r *Recommender) GetPodGroups() []PodGroup {
	result := []PodGroup{}
	nsVars := []utils.Var{{Name: "namespace", Value: r.Namespace}}
	//get Pod groups sts/dep/daemonset/cronjobs/spark jobs
	result = append(result, r.getPodGroupKind(cron, queryCron, nsVars)...)
	result = append(result, r.getPodGroupKind(sparkDrivers, queryDriver, nsVars)...)
	result = append(result, r.getPodGroupKind(sparkExecutors, queryExecutor, nsVars)...)
	result = append(result, r.getPodGroupKind(sts, querySts, nsVars)...)
	// result = append(result, r.getPodGroupKind(rs, queryRs, nsVars)...)
	result = append(result, r.getPodGroupKind(ds, queryDs, nsVars)...)
	result = append(result, r.getPodGroupKind(dep, queryDep, nsVars)...)

	return result
}

func (r *Recommender) getPodGroupKind(kind string, query string, vars []utils.Var) []PodGroup {
	result := []PodGroup{}
	query, err := utils.SubstVars(query, vars)
	if err != nil {
		log.Error("Error subst Vars:", err)
		return result
	}
	data, err := utils.PromQuery(r.PromURL, query, vars)
	if err != nil {
		log.Error("PromQL Instant query wrong for ", query, " err ", err)
	} else {
		log.Info("Query PodGroups for ", query)
		vectorVal, ok := data.(model.Vector)
		if !ok {
			log.Error("Error converting to Vector for query ", query)
			return result
		}
		for _, elem := range vectorVal {
			result = append(result, PodGroup{Kind: kind, Name: string(elem.Metric[model.LabelName(kind)]), Namespace: string(elem.Metric["namespace"]), Count: int(elem.Value), Suffix: r.getPodGroupSuffix(kind)})
		}
	}
	return result
}

// getPodGroupSuffix returns the suffix for the pod group
func (r *Recommender) getPodGroupSuffix(kind string) string {
	switch kind {
	case sts:
		return "-\\\\d+"
	case rs:
		return "-\\\\w+-\\\\w+"
	case ds:
		return "-\\\\w+"
	case dep:
		return "-\\\\w+-\\\\w+"
	case cron:
		return "-\\\\w+-\\\\w+"
	case sparkDrivers:
		return ".*"
	case sparkExecutors:
		return "-\\\\w+-exec-\\\\d+"
	}
	log.Error("Unknown Pod Group Kind ", kind)
	return ""
}
