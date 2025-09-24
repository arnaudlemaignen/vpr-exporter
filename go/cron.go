package main

import (
	"sort"
	"strconv"
	"time"
	"vpr/pkg/rec"

	log "github.com/sirupsen/logrus"
)

func getData() {
	var result []rec.Recommendation

	log.Info("Start Recommender")
	r := rec.NewRecommender(limitAliases)
	r.ShowConfig()

	//1. get Pod groups sts/dep/daemonset
	durationLimit := time.Duration(0)
	durationUsage := time.Duration(0)
	durationJVMUsage := time.Duration(0)
	durationRecommendation := time.Duration(0)
	timeStart := time.Now()

	log.Info()
	podGroups := r.GetPodGroups()
	log.Info("Found ", len(podGroups), " PodGroups in ", time.Since(timeStart))
	//2. calculate req/limit for each pod group
	for i, podGroup := range podGroups {
		//get limits and requests for each pod group
		timeLimitInfo := time.Now()
		limits := r.GetPodGroupLimits(podGroup.Namespace, podGroup.Name, podGroup.Suffix)
		durationLimit += time.Since(timeLimitInfo)

		//get usage for each pod group
		timeUsageInfo := time.Now()
		usage := r.GetPodGroupUsage(podGroup.Namespace, podGroup.Name, podGroup.Suffix)
		durationUsage += time.Since(timeUsageInfo)

		//get jvm usage for each pod group
		timeJVMInfo := time.Now()
		jvmUsage := r.GetPodGroupJVMUsage(podGroup.Namespace, podGroup.Name, podGroup.Suffix)
		durationJVMUsage += time.Since(timeJVMInfo)

		//get recommendations for each pod group
		timeRecInfo := time.Now()
		recs := r.GenRecommendation(podGroup, usage, jvmUsage, limits)
		durationRecommendation += time.Since(timeRecInfo)

		for _, rec := range recs {
			log.Trace(rec)
		}
		log.Info(strconv.FormatFloat((float64(i)+1.0)*100.0/float64(len(podGroups)), 'f', 1, 64), " % completion => PodGroup ", i, " / ", len(podGroups), " : ", podGroup.Kind, " ", podGroup.Name)
		result = append(result, recs...)
	}

	// Write CSV results with sorting
	// Sort results by GainMemReqMB in descending order
	sort.Slice(result, func(i, j int) bool {
		return result[i].GainMemReqMB > result[j].GainMemReqMB
	})
	r.GenCSVRecommendations(result)

	// Write helm-value results with filtering the dim helm values
	r.GenYAMLLimitRecommendations(result)
	//calculate total optimization
	cpu, mem := r.CalculateMaxOptimization(result)

	timeFinal := time.Now()
	log.Info("VPR recommendations (CPU: ", cpu, " vCPUs Mem: ", mem, " GiB optimizations) generated in ", timeFinal.Sub(timeStart), " details (Limit ", durationLimit, " Usage ", durationUsage, " JVM Usage ", durationJVMUsage, " Reco ", durationRecommendation, ")")
}
