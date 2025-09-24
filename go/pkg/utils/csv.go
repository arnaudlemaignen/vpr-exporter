package utils

import (
	"encoding/csv"
	"os"
	"strconv"

	log "github.com/sirupsen/logrus"
)

// GenCSV to generate CSV files
func GenCSV(filePath string, data [][]string) {
	csvFile, err := os.Create(filePath)
	if err != nil {
		log.Error("failed creating file:", filePath, " err ", err)
	}
	csvwriter := csv.NewWriter(csvFile)

	for _, row := range data {
		_ = csvwriter.Write(row)
	}
	csvwriter.Flush()
	csvFile.Close()
}

// PodContainerExtraParams struct for pod container and limit alias
type PodContainerExtraParams struct {
	Pod                string
	Container          string
	LimitAlias         string
	HelmValueFileName  string
	UntouchMemoryLimit bool
	ExtraMemoryMargin  int
}

// ReadLimitAliasCSVFile to read the container limit aliases from a CSV file
// pod_name,container_name,limit_alias
func ReadLimitAliasCSVFile() ([]PodContainerExtraParams, error) {
	filename := "resources/container_limit_aliases.csv"
	config := make([]PodContainerExtraParams, 0)

	if len(filename) == 0 {
		return config, nil
	}
	file, err := os.Open(filename)
	if err != nil {
		log.Error("ReadLimitAliasCSVFile error open file ", filename, " err ", err)
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		log.Error("ReadLimitAliasCSVFile error reading file ", filename, " err ", err)
		return nil, err
	}

	for _, record := range records {
		if len(record) < 4 {
			log.Warn("ReadLimitAliasCSVFile record has less than 4 fields, skipping: ", record)
			continue
		}
		extraMemMargingPer, _ := strconv.Atoi(record[5])
		// Create a new PodContainerLimitAlias struct and append it to the config slice
		alias := PodContainerExtraParams{
			Pod:                record[0],
			Container:          record[1],
			LimitAlias:         record[2],
			HelmValueFileName:  record[3],
			UntouchMemoryLimit: record[4] == "true",
			ExtraMemoryMargin:  extraMemMargingPer,
		}
		config = append(config, alias)
	}

	return config, nil
}
