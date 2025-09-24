package rec

import (
	"math"
	"os"
	"strconv"
	"strings"
	"vpr/pkg/types"
	"vpr/pkg/utils"

	"sort"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

const (
	// OutPathCsvRecommendations is the path to the CSV file with limits recommendations
	OutPathCsvRecommendations = types.DataPath + "recommendations.csv"
)

// ConfigMap is a struct to generate a ConfigMap
type ConfigMap struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
	} `yaml:"metadata"`
	Data map[string]string `yaml:"data"`
}

// GenCSVRecommendations func to generate a CSV file with all Recommendations elements as output
func (r *Recommender) GenCSVRecommendations(rec []Recommendation) {
	csvData := [][]string{{"Namespace", "Kind", "PodGroupName", "Replicas", "ContainerName", "LimitAlias",
		"CPUReqM", "MemReqMB", "CPULimitM", "MemLimitMB", "NewCPUReqM", "NewMemReqMB", "NewMemLimitMB", "GainCPUReqM", "GainMemReqMB",
		"CPUMinM", "CPUMeanM", "CPUPercentileM", "CPUMaxM", "MemMinMB", "MemMeanMB", "MemPercentileMB", "MemMaxMB",
		"JVMYoungGenMB", "JVMYoungGenMinMB", "JVMYoungGenMaxAfterGCMB", "JVMYoungGenMaxMB", "JVMOldGenMinMB", "JVMOldGenMaxAfterFullGCMB", "JVMOldGenMaxMB", "JVMXmxPercent", "JVMAllocationStalls"}}
	for _, elem := range rec {
		csvData = append(csvData, [][]string{{
			elem.Namespace,
			elem.Kind,
			elem.PodGroupName,
			strconv.Itoa(elem.Replicas),
			elem.ContainerName,
			elem.LimitAlias,
			strconv.FormatFloat(elem.CPUReqM, 'f', 0, 64),
			strconv.FormatFloat(elem.MemReqMB, 'f', 0, 64),
			strconv.FormatFloat(elem.CPULimitM, 'f', 0, 64),
			strconv.FormatFloat(elem.MemLimitMB, 'f', 0, 64),
			strconv.FormatFloat(elem.NewCPUReqM, 'f', 0, 64),
			strconv.FormatFloat(elem.NewMemReqMB, 'f', 0, 64),
			strconv.FormatFloat(elem.NewMemLimitMB, 'f', 0, 64),
			strconv.FormatFloat(elem.GainCPUReqM, 'f', 0, 64),
			strconv.FormatFloat(elem.GainMemReqMB, 'f', 0, 64),
			strconv.FormatFloat(elem.CPUMinM, 'f', 0, 64),
			strconv.FormatFloat(elem.CPUMeanM, 'f', 0, 64),
			strconv.FormatFloat(elem.CPUPercentileM, 'f', 0, 64),
			strconv.FormatFloat(elem.CPUMaxM, 'f', 0, 64),
			strconv.FormatFloat(elem.MemMinMB, 'f', 0, 64),
			strconv.FormatFloat(elem.MemMeanMB, 'f', 0, 64),
			strconv.FormatFloat(elem.MemPercentileMB, 'f', 0, 64),
			strconv.FormatFloat(elem.MemMaxMB, 'f', 0, 64),
			strconv.FormatFloat(elem.JVMYoungGenMB, 'f', 0, 64),
			strconv.FormatFloat(elem.JVMYoungGenMinMB, 'f', 0, 64),
			strconv.FormatFloat(elem.JVMYoungGenMaxAfterGCMB, 'f', 0, 64),
			strconv.FormatFloat(elem.JVMYoungGenMaxMB, 'f', 0, 64),
			strconv.FormatFloat(elem.JVMOldGenMinMB, 'f', 0, 64),
			strconv.FormatFloat(elem.JVMOldGenMaxAfterFullGCMB, 'f', 0, 64),
			strconv.FormatFloat(elem.JVMOldGenMaxMB, 'f', 0, 64),
			strconv.FormatFloat(elem.JVMXmxPercent, 'f', 0, 64),
			strconv.Itoa(elem.JVMAllocationStalls),
		}}...)
	}

	utils.GenCSV(OutPathCsvRecommendations, csvData)
}

// CalculateMaxOptimization calculates the maximum possible optimization on CPU and Memory.
func (r *Recommender) CalculateMaxOptimization(result []Recommendation) (string, string) {
	totalGainCPUReqM := 0.0
	totalGainMemReqMB := 0.0
	for _, rec := range result {
		if rec.LimitAlias != "NA" {
			if rec.GainCPUReqM > 50.0 {
				totalGainCPUReqM += rec.GainCPUReqM
			}
			if rec.GainMemReqMB > 100.0 {
				totalGainMemReqMB += rec.GainMemReqMB
			}
		}
	}
	return strconv.FormatFloat(totalGainCPUReqM, 'f', 0, 64), strconv.FormatFloat(totalGainMemReqMB/1024.0, 'f', 0, 64)
}

// GenYAMLLimitRecommendations func to generate a YAML file with all Recommendations elements as output
func (r *Recommender) GenYAMLLimitRecommendations(rec []Recommendation) {
	//reorder the recommendations by LimitAlias
	SortRecommendationsByLimitAlias(rec)

	helmValueFiles := make(map[string][]Recommendation)

	//group the recommendations by HelmValueFileName
	for _, elem := range rec {
		if elem.HelmValueFileName == "" {
			log.Warn("HelmValueFileName is empty for ", elem.PodGroupName, " skipping recommendation")
			continue
		}
		helmValueFiles[elem.HelmValueFileName] = append(helmValueFiles[elem.HelmValueFileName], elem)
	}

	//generate a helm values file for each HelmValueFileName
	for helmValueFileName, recommendations := range helmValueFiles {
		r.genHelmValuesFile(helmValueFileName, recommendations)
	}
}

func (r *Recommender) genHelmValuesFile(helmValuesName string, rec []Recommendation) {
	path := types.DataPath + helmValuesName + ".yaml"

	//remove duplicates from the recommendations
	// this is needed to avoid duplicates in the helm values file
	// as the same PodGroupName can have multiple recommendations with different LimitAlias
	// e.g. kibana and kibana-app has the same "kibana" limit alias
	newRec := removeDuplicates(rec)

	configMap := ConfigMap{
		APIVersion: "v1",
		Kind:       "ConfigMap",
	}
	configMap.Metadata.Name = helmValuesName
	configMap.Data = map[string]string{
		"values.yaml": r.genDimValues(newRec),
	}

	// Marshal the recommendations into YAML format
	yamlData, err := yaml.Marshal(configMap)
	if err != nil {
		log.Error("Error marshaling ConfigMap to YAML: ", err)
	}

	// Write the YAML data to a file
	file, err := os.Create(path)
	if err != nil {
		log.Error("Error creating YAML file: ", err)
	}
	defer file.Close()

	_, err = file.Write(yamlData)
	if err != nil {
		log.Error("Error writing to YAML file: ", err)
	}

	log.Info("Generated helm values file: ", helmValuesName, " with ", len(newRec), " recommendations")
}

// remove duplicates from the recommendations
// removeDuplicates removes duplicates from the recommendations
// it takes a slice of Recommendation as input
// and returns a slice of Recommendation without duplicates
func removeDuplicates(recs []Recommendation) []Recommendation {
	var uniqueRecs []Recommendation
	previousIndex := 1
	previousLimitAlias := ""
	previousPodGroupName := ""
	previousGainCPU := math.MaxFloat64
	previousGainMem := math.MaxFloat64
	previousReplicas := 0
	previousCPUReq := -1.0
	previousMemReq := -1.0
	previousNewMemReq := 1.0
	previousNewMemLimit := -1.0
	previousNewCPUReq := -1.0

	for _, rec := range recs {
		if rec.LimitAlias != "NA" {
			// Check if the LimitAlias is not "NA" and not empty
			if rec.LimitAlias == previousLimitAlias {
				newRec := Recommendation{}
				newRec.PodGroupName = previousPodGroupName + "_" + rec.PodGroupName
				newRec.Replicas = previousReplicas + rec.Replicas
				newRec.LimitAlias = rec.LimitAlias
				newRec.Namespace = rec.Namespace
				newRec.Kind = rec.Kind
				newRec.ContainerName = rec.ContainerName
				//we are looking for keeping only the lowest GainCPUReqM and GainMemReqMB to avoid CPU/Mem shortage
				if rec.GainCPUReqM < previousGainCPU {
					newRec.GainCPUReqM = (rec.CPUReqM - rec.NewCPUReqM) * float64(newRec.Replicas)
					newRec.CPUReqM = rec.CPUReqM
					newRec.NewCPUReqM = rec.NewCPUReqM
				} else {
					newRec.GainCPUReqM = (previousCPUReq - previousNewCPUReq) * float64(newRec.Replicas)
					newRec.CPUReqM = previousCPUReq
					newRec.NewCPUReqM = previousNewCPUReq
				}
				if rec.GainMemReqMB < previousGainMem {
					newRec.GainMemReqMB = (rec.MemReqMB - rec.NewMemReqMB) * float64(newRec.Replicas)
					newRec.MemReqMB = rec.MemReqMB
					newRec.NewMemReqMB = rec.NewMemReqMB
					newRec.NewMemLimitMB = rec.NewMemLimitMB
				} else {
					newRec.GainMemReqMB = (previousMemReq - previousNewMemReq) * float64(newRec.Replicas)
					newRec.MemReqMB = previousMemReq
					newRec.NewMemReqMB = previousNewMemReq
					newRec.NewMemLimitMB = previousNewMemLimit
				}
				uniqueRecs = removeIndex(uniqueRecs, previousIndex)
				uniqueRecs = append(uniqueRecs, newRec)

				previousIndex = len(uniqueRecs) - 1
				previousLimitAlias = newRec.LimitAlias
				previousPodGroupName = newRec.PodGroupName
				previousReplicas = newRec.Replicas
				previousGainCPU = newRec.GainCPUReqM
				previousGainMem = newRec.GainMemReqMB
				previousNewMemReq = newRec.NewMemReqMB
				previousNewMemLimit = newRec.NewMemLimitMB
				previousNewCPUReq = newRec.NewCPUReqM
			} else {
				uniqueRecs = append(uniqueRecs, rec)
				previousIndex = len(uniqueRecs) - 1
				previousLimitAlias = rec.LimitAlias
				previousPodGroupName = rec.PodGroupName
				previousReplicas = rec.Replicas
				previousGainCPU = rec.GainCPUReqM
				previousGainMem = rec.GainMemReqMB
				previousNewMemReq = rec.NewMemReqMB
				previousNewMemLimit = rec.NewMemLimitMB
				previousNewCPUReq = rec.NewCPUReqM
			}
		} else {
			uniqueRecs = append(uniqueRecs, rec)
			previousIndex = len(uniqueRecs) - 1
			previousLimitAlias = rec.LimitAlias
			previousPodGroupName = rec.PodGroupName
			previousReplicas = rec.Replicas
			previousGainCPU = rec.GainCPUReqM
			previousGainMem = rec.GainMemReqMB
			previousNewMemReq = rec.NewMemReqMB
			previousNewMemLimit = rec.NewMemLimitMB
			previousNewCPUReq = rec.NewCPUReqM
		}
	}
	return uniqueRecs
}

// generate the dim values in the yaml file
// genDimValues generates the dim values in the yaml file
// it takes a slice of Recommendation as input
// and generates the dim values in the yaml file
// Several use cases are supported:
// - Pod with single container (e.g. Values.res.prometheus.requests.cpu)
// - Pod with multiple containers (e.g. Values.res.prometheus.web.requests.cpu)

func (r *Recommender) genDimValues(rec []Recommendation) string {
	var sb strings.Builder
	previousLevel0 := ""
	previousLevel1 := ""
	previousLevel2 := ""
	gainCPUReq := 0.0
	gainMemReq := 0.0

	//generate the date string for now shown as YYYY-MM-DD
	// this is used to show the date when the recommendations were generated
	sb.WriteString("# VPR recommendations\n")

	for _, elem := range rec {
		if elem.Namespace == r.Namespace || r.Namespace == ".*" {
			//we dont bend down to pick up pennies
			//at least 50 m or 100 MiB gain and only if LimitAlias is known
			if (elem.GainCPUReqM > 50.0 || elem.GainMemReqMB > 100.0) && elem.LimitAlias != "NA" {
				limitLevel := strings.Split(elem.LimitAlias, ".")
				if len(limitLevel) < 2 || len(limitLevel) > 3 {
					log.Warn("LimitAlias ", elem.LimitAlias, " for ", elem.PodGroupName, " is not valid, skipping recommendation")
					continue
				}
				//Write level 1
				if limitLevel[0] != previousLevel0 {
					sb.WriteString(limitLevel[0] + ":\n")
					previousLevel0 = limitLevel[0]
				}

				//Check if level 3 is present
				if len(limitLevel) == 3 {
					//Write level 2
					if limitLevel[1] != previousLevel1 {
						sb.WriteString(strings.Repeat("  ", 1) + limitLevel[1] + ":\n")
						previousLevel1 = limitLevel[1]
					}
					if limitLevel[2] != previousLevel2 {
						sb.WriteString(strings.Repeat("  ", 2) + limitLevel[2] + ":\n")
						previousLevel2 = limitLevel[2]
					}
					sb.WriteString(strings.Repeat("  ", len(limitLevel)) + "# " + elem.Namespace + " | " + elem.PodGroupName + " | " + elem.ContainerName + "\n")
					sb.WriteString(strings.Repeat("  ", len(limitLevel)) + "requests:\n")
					if elem.GainCPUReqM > 50.0 {
						sb.WriteString(strings.Repeat("  ", 1+len(limitLevel)) + "cpu: " + strconv.FormatFloat(elem.NewCPUReqM, 'f', 0, 64) + "m")
						sb.WriteString(" # Gain " + strconv.FormatFloat(elem.GainCPUReqM, 'f', 0, 64) + "m\n")
						gainCPUReq += elem.GainCPUReqM
					}
					if elem.GainMemReqMB > 100.0 {
						sb.WriteString(strings.Repeat("  ", 1+len(limitLevel)) + "memory: " + strconv.FormatFloat(elem.NewMemReqMB, 'f', 0, 64) + "Mi")
						sb.WriteString(" # Gain " + strconv.FormatFloat(elem.GainMemReqMB, 'f', 0, 64) + " Mi\n")
						sb.WriteString(strings.Repeat("  ", len(limitLevel)) + "limits:\n")
						sb.WriteString(strings.Repeat("  ", 1+len(limitLevel)) + "memory: " + strconv.FormatFloat(elem.NewMemLimitMB, 'f', 0, 64) + "Mi\n")
						gainMemReq += elem.GainMemReqMB
					}
				} else {
					//Write level 2
					if limitLevel[1] != previousLevel1 {
						sb.WriteString(strings.Repeat("  ", 1) + limitLevel[1] + ":\n")
						previousLevel1 = limitLevel[1]
					}
					sb.WriteString(strings.Repeat("  ", len(limitLevel)) + "# " + elem.Namespace + " | " + elem.PodGroupName + " | " + elem.ContainerName + "\n")
					sb.WriteString(strings.Repeat("  ", len(limitLevel)) + "requests:\n")
					if elem.GainCPUReqM > 50.0 {
						sb.WriteString(strings.Repeat("  ", 1+len(limitLevel)) + "cpu: " + strconv.FormatFloat(elem.NewCPUReqM, 'f', 0, 64) + "m")
						sb.WriteString(" # Gain " + strconv.FormatFloat(elem.GainCPUReqM, 'f', 0, 64) + " m\n")
						gainCPUReq += elem.GainCPUReqM
					}
					if elem.GainMemReqMB > 100.0 {
						sb.WriteString(strings.Repeat("  ", 1+len(limitLevel)) + "memory: " + strconv.FormatFloat(elem.NewMemReqMB, 'f', 0, 64) + "Mi")
						sb.WriteString(" # Gain " + strconv.FormatFloat(elem.GainMemReqMB, 'f', 0, 64) + " Mi\n")
						sb.WriteString(strings.Repeat("  ", len(limitLevel)) + "limits:\n")
						sb.WriteString(strings.Repeat("  ", 1+len(limitLevel)) + "memory: " + strconv.FormatFloat(elem.NewMemLimitMB, 'f', 0, 64) + "Mi\n")
						gainMemReq += elem.GainMemReqMB
					}
				}
			}
		}
	}
	sb.WriteString("# Overall gain on CPU req " + strconv.FormatFloat(gainCPUReq, 'f', 0, 64) + " m | Mem req " + strconv.FormatFloat(gainMemReq, 'f', 0, 64) + " Mi\n")
	return sb.String()
}

func replaceDashByUnderscore(s string) string {
	return strings.ReplaceAll(s, "-", "_")
}

// SortRecommendationsByLimitAlias sorts a slice of Recommendation by LimitAlias (ascending)
func SortRecommendationsByLimitAlias(recs []Recommendation) {
	sort.Slice(recs, func(i, j int) bool {
		return recs[i].LimitAlias < recs[j].LimitAlias
	})
}

func removeIndex(s []Recommendation, index int) []Recommendation {
	// Simple case: remove last element
	if index == len(s)-1 {
		return s[:len(s)-1]
	}

	// General case: remove element at index
	return append(s[:index], s[index+1:]...)
}
