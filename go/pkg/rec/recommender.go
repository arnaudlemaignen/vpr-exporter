package rec

import (
	"time"
	"vpr/pkg/utils"

	log "github.com/sirupsen/logrus"
)

// Recommender is a struct with all the necessary fields for the Recommender
type Recommender struct {
	PromURL, Namespace                                                                                                                                              string
	History, Interval                                                                                                                                               time.Duration
	PodMinCPUMillicores, PodMinMemoryMb, TargetCPUPercentile, TargetMemPercentile, TargetMemLimitToReqPercent, TargetMemOldGenUsagePercent, TargetMemStaticMaxRatio float64
	ExtraParams                                                                                                                                                     []utils.PodContainerExtraParams
}

// NewRecommender creates a new Recommender
func NewRecommender(extraParams []utils.PodContainerExtraParams) *Recommender {
	return &Recommender{
		PromURL:                     assemblePrometheusURL(),
		Namespace:                   utils.GetStringEnv("NAMESPACE", ".*"),
		History:                     utils.GetDurationEnv("HISTORY", 7*24*time.Hour),
		Interval:                    utils.GetDurationEnv("INTERVAL", time.Minute),
		PodMinCPUMillicores:         utils.GetFloat64Env("POD_MIN_CPU_M", 5),
		PodMinMemoryMb:              utils.GetFloat64Env("POD_MIN_MEM_MB", 50),
		TargetCPUPercentile:         utils.GetFloat64Env("TARGET_CPU_PERCENTILE", 90),
		TargetMemPercentile:         utils.GetFloat64Env("TARGET_MEM_PERCENTILE", 90),
		TargetMemLimitToReqPercent:  utils.GetFloat64Env("TARGET_MEM_LIMIT_TO_REQ_PERCENT", 85),
		TargetMemOldGenUsagePercent: utils.GetFloat64Env("TARGET_MEM_OLD_GEN_USAGE_PERCENT", 65),
		TargetMemStaticMaxRatio:     utils.GetFloat64Env("TARGET_MEM_STATIC_MAX_RATIO", 3),
		ExtraParams:                 extraParams,
	}
}

func assemblePrometheusURL() string {
	promURL := "http://prometheus:9090"
	promHTTPSchema := utils.GetStringEnv("PROMETHEUS_HTTP_SCHEMA", "http")
	promEndpoint := utils.GetStringEnv("PROMETHEUS_ENDPOINT", "prometheus:8080")
	promUser := utils.GetStringEnv("PROMETHEUS_AUTH_USER", "login")
	promPwd := utils.GetStringEnv("PROMETHEUS_AUTH_PWD", "pwd")
	promLogin := ""
	if promUser == "" || promPwd == "" {
		log.Info("PROMETHEUS_AUTH_USER and or PROMETHEUS_AUTH_PWD were not set, will not use basic auth.")
		promURL = promHTTPSchema + "://" + promEndpoint
	} else {
		// http://user:pass@localhost/ to use basic auth
		promLogin = promUser + ":" + promPwd
		promURL = promHTTPSchema + "://" + promLogin + "@" + promEndpoint
	}

	return promURL
}

// ShowConfig shows the configuration of the Recommender
func (r *Recommender) ShowConfig() {
	log.Infof("Prometheus URL: %s", r.PromURL)
	log.Infof("Namespace: %s", r.Namespace)
	log.Infof("History: %s", r.History)
	log.Infof("Interval: %s", r.Interval)
	log.Infof("PodMinCPUMillicores: %f", r.PodMinCPUMillicores)
	log.Infof("PodMinMemoryMb: %f", r.PodMinMemoryMb)
	log.Infof("TargetCPUPercentile: %f", r.TargetCPUPercentile)
	log.Infof("TargetMemPercentile: %f", r.TargetMemPercentile)
	log.Infof("TargetMemLimitToReqPercent: %f", r.TargetMemLimitToReqPercent)
	log.Infof("TargetMemOldGenUsagePercent: %f", r.TargetMemOldGenUsagePercent)
}
