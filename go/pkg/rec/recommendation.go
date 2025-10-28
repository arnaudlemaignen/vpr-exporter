package rec

import (
	"math"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

// Recommendation is a struct with all the necessary fields for the recommendation
type Recommendation struct {
	Namespace         string
	Kind              string
	PodGroupName      string
	Replicas          int
	ContainerName     string
	LimitAlias        string
	HelmValueFileName string
	CPUReqM           float64
	MemReqMB          float64
	CPULimitM         float64
	MemLimitMB        float64
	NewCPUReqM        float64
	NewMemReqMB       float64
	NewMemLimitMB     float64
	GainCPUReqM       float64
	GainMemReqMB      float64
	//details
	CPUMinM         float64
	CPUMeanM        float64
	CPUPercentileM  float64
	CPUMaxM         float64
	MemMinMB        float64
	MemMeanMB       float64
	MemPercentileMB float64
	MemMaxMB        float64
	//JVM
	JVMYoungGenMB             float64
	JVMOldGenMinMB            float64
	JVMOldGenMaxMB            float64
	JVMOldGenMaxAfterFullGCMB float64
	JVMXmxPercent             float64
	JVMAllocationStalls       int
	//TRIAL
	JVMYoungGenMinMB        float64
	JVMYoungGenMaxAfterGCMB float64
	JVMYoungGenMaxMB        float64
}

// GenRecommendation produces a recommendation based on the usage
func (r *Recommender) GenRecommendation(podGroup PodGroup, usage map[string]ContainerUsage, jvmUsage map[string]JVMContainerUsage, limits map[string]ContainerLimits) []Recommendation {
	result := []Recommendation{}
	//only usage exists, a Bergson concept (only the movement exists)
	for containerName, elem := range usage {
		limitAlias, helmValueFileName, untouchMemoryLimit, extraMemoryMargin := r.findExtraParams(podGroup, containerName)

		// all params that are for sure
		c := Recommendation{
			Namespace:     podGroup.Namespace,
			Kind:          podGroup.Kind,
			PodGroupName:  podGroup.Name,
			Replicas:      podGroup.Count,
			ContainerName: containerName,
			LimitAlias:    limitAlias,
			//helmValueFileName is not used in the recommendation but can be used to generate helm value files with the
			HelmValueFileName: helmValueFileName,

			//details
			CPUMinM:         elem.CPUUsageM.Min,
			CPUMeanM:        elem.CPUUsageM.Mean,
			CPUPercentileM:  elem.CPUUsageM.Percentile,
			CPUMaxM:         elem.CPUUsageM.Max,
			MemMinMB:        elem.MemUsageMB.Min,
			MemMeanMB:       elem.MemUsageMB.Mean,
			MemPercentileMB: elem.MemUsageMB.Percentile,
			MemMaxMB:        elem.MemUsageMB.Max,
		}
		// Check if the containerName exists in the limits map
		if val, ok := limits[containerName]; ok {
			c.CPUReqM = val.CPUReqM
			c.MemReqMB = val.MemReqMB
			c.CPULimitM = val.CPULimitM
			c.MemLimitMB = val.MemLimitMB
		} else {
			log.Warn("No limits found for container ", containerName, " in pod group ", podGroup.Name, " will recommend limits based on usage")
		}
		//CPU Recommendation Req & NO CPU limit
		if elem.CPUUsageM.Percentile > r.PodMinCPUMillicores {
			c.NewCPUReqM = elem.CPUUsageM.Percentile
		} else {
			c.NewCPUReqM = r.PodMinCPUMillicores
		}

		if val, ok := jvmUsage[containerName]; ok && val.OldGenUsageMB.Max > 0 && c.MemLimitMB > 0 {
			//WE WILL RECOMMEND Mem REQ and Mem LIMIT based on JVM only if JVM metrics are available
			//calculate JVM Xmx
			c.JVMXmxPercent = (val.YoungPoolMB + val.OldPoolMB) * 100.0 / c.MemLimitMB
			if c.JVMXmxPercent < 1.0 {
				log.Warn("Xmx% is incorrect, recommendation will be skipped for pod ", podGroup.Name, "  container ", containerName, " instant OldPoolMB ", val.OldPoolMB)
				continue
			}
			c.JVMYoungGenMB = val.YoungGenSizeMB
			c.JVMAllocationStalls = val.AllocationStall
			c.JVMOldGenMinMB = val.OldGenUsageMB.Min
			c.JVMOldGenMaxMB = val.OldGenUsageMB.Max
			c.JVMOldGenMaxAfterFullGCMB = val.OldGenUsageAfterGcMB
			c.JVMYoungGenMinMB = val.YoungGenUsageMB.Min
			c.JVMYoungGenMaxAfterGCMB = val.YoungGenUsageMB.MaxAfterGC
			c.JVMYoungGenMaxMB = val.YoungGenUsageMB.Max
			// new Xmx  = Young + max(max Old Gen used after GC/.65, 3*static memory)
			// new Limit = new Xmx / Xmx%
			// new Req   = new Limit * 85%
			//static memory is the Heap memory containing JVM metadata which is the baseline of the JVM graph (ie the min of the OldGen usage)
			//transaction memory is the Heap memory that cannot be garbage collected during transactions (ie the max after full GC of the OldGen usage)
			maxTransactionVsStaticMemory := math.Max(c.JVMOldGenMaxAfterFullGCMB*100.0/r.TargetMemOldGenUsagePercent, r.TargetMemStaticMaxRatio*c.JVMOldGenMinMB)

			//Old algo
			//Cons does not work well with Java 24 with ZGC Young Generation which is sometimes = to Xmx and with ElasticSearch which has no Young Generation value
			// newXmx := c.JVMYoungGenMB + maxTransactionVsStaticMemory
			//New algo
			newXmx := c.JVMYoungGenMaxAfterGCMB + maxTransactionVsStaticMemory
			newLimit := newXmx * 100.0 / c.JVMXmxPercent
			//extra protective measure
			//for Java processes the min Xmx is 512MB, so we will not recommend less than that
			//and if the new Limit is between 300MB and 1GB, we will recommend 1GB
			if newLimit < 300.0 {
				newLimit = 512.0
			} else if newLimit < 1024.0 {
				newLimit = 1024.0
			}

			c.NewMemLimitMB = newLimit
			c.NewMemReqMB = c.NewMemLimitMB * r.TargetMemLimitToReqPercent / 100.0
			//only applied if extraMemoryMargin > 0 and for the limit (req remains the same)
			if extraMemoryMargin > 0 {
				c.NewMemLimitMB = float64(100+extraMemoryMargin) * c.NewMemLimitMB / 100.0
			}
		} else {
			//WE WILL RECOMMEND Mem REQ and Mem LIMIT based on USAGE
			if elem.MemUsageMB.Percentile > r.PodMinMemoryMb {
				c.NewMemReqMB = elem.MemUsageMB.Percentile
			} else {
				c.NewMemReqMB = r.PodMinMemoryMb
			}
			if untouchMemoryLimit {
				log.Info("Untouching memory limit for pod ", podGroup.Name, " container ", containerName, " using LimitAlias ", limitAlias)
				c.NewMemLimitMB = c.MemLimitMB //keep the current limit
			} else {
				log.Debug("Recommending memory limit for pod ", podGroup.Name, " container ", containerName, " using LimitAlias ", limitAlias)
				//calculate Mem Limit as the max mem usage + some buffer
				c.NewMemLimitMB = elem.MemUsageMB.Max * 100.0 / r.TargetMemLimitToReqPercent
				//harness the Mem Limit to be at least the PodMinMemoryMb
				if c.NewMemLimitMB < r.PodMinMemoryMb {
					c.NewMemLimitMB = r.PodMinMemoryMb
				}
				//only applied if extraMemoryMargin > 0 and for the limit (req remains the same)
				if extraMemoryMargin > 0 {
					c.NewMemLimitMB = float64(100+extraMemoryMargin) * c.NewMemLimitMB / 100.0
				}
			}
		}

		//calculate Gain
		c.GainCPUReqM = float64(podGroup.Count) * (c.CPUReqM - c.NewCPUReqM)
		c.GainMemReqMB = float64(podGroup.Count) * (c.MemReqMB - c.NewMemReqMB)
		result = append(result, c)
	}
	return result
}

func (r *Recommender) findExtraParams(podGroup PodGroup, containerName string) (string, string, bool, int) {
	for _, extra := range r.ExtraParams {
		//if the Pod name respect the pod name regex
		//^ is the start of the string, $ is the end of the string
		matchPodGroup, _ := regexp.MatchString("^"+extra.Pod+"$", podGroup.Name)
		matchContainer, _ := regexp.MatchString("^"+extra.Container+"$", containerName)
		if matchPodGroup && matchContainer {
			definitiveHelmValueFileName := extra.HelmValueFileName
			//check if the HelmValueFileName contains a $1 pattern to be replaced by the PodGroup regex capture group
			definitiveHelmValueFileName = replaceCaptureGroup(extra.Pod, podGroup.Name, definitiveHelmValueFileName)

			// re := regexp.MustCompile(extra.Pod)
			// matches := re.FindStringSubmatch(podGroup.Name)
			// if len(matches) > 1 {
			// 	//there is at least one capture group
			// 	//replace $1 with the first capture group
			// 	definitiveHelmValueFileName = re.ReplaceAllString(definitiveHelmValueFileName, matches[1])
			// }
			definitiveHelmValueFileName = "helm-values-" + definitiveHelmValueFileName
			return extra.LimitAlias, definitiveHelmValueFileName, extra.UntouchMemoryLimit, extra.ExtraMemoryMargin //return the limit alias
		}
	}
	return "NA", "", false, 0 //if no match found, return NA and false
}

func replaceCaptureGroup(pattern, str, replacement string) string {
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(str)
	if len(matches) > 1 {
		//there is at least one capture group
		//replace $1 with the first capture group
		return strings.ReplaceAll(replacement, "$1", matches[1])
	}
	return replacement
}
