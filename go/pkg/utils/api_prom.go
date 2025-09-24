package utils

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"regexp"

	// "os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	log "github.com/sirupsen/logrus"
)

// Var struct
type Var struct {
	Name  string
	Value string
}

// PromQuery INSTANT queries
func PromQuery(endpoint string, query string, vars []Var) (model.Value, error) {
	// Create a custom HTTP client that skips certificate verification
	customClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	client, err := api.NewClient(api.Config{
		Address: endpoint,
		Client:  customClient,
	})
	if err != nil {
		log.Error("Error creating client:", err)
		return nil, err
	}

	v1api := v1.NewAPI(client)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result, warnings, err := v1api.Query(ctx, query, time.Now())
	if err != nil {
		log.Error("Error querying Prometheus:", err)
		return nil, err
	}
	if len(warnings) > 0 {
		log.Warn("Warnings:", warnings)
	}
	log.Trace("Result:", result)
	return result, nil
}

// PromQueryRange RANGE queries
func PromQueryRange(endpoint string, query string, vars []Var, start time.Time, end time.Time, step time.Duration) (model.Value, error) {
	customClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	client, err := api.NewClient(api.Config{
		Address: endpoint,
		Client:  customClient,
	})
	if err != nil {
		log.Error("Error creating client:", err)
		return nil, err
	}

	v1api := v1.NewAPI(client)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	r := v1.Range{
		Start: start,
		End:   end,
		Step:  step,
	}
	result, warnings, err := v1api.QueryRange(ctx, query, r)
	if err != nil {
		log.Error("Error querying Prometheus:", err)
		return nil, err
	}
	if len(warnings) > 0 {
		log.Warn("Warnings:", warnings)
	}
	log.Trace("Result:", result)
	return result, nil
}

// PromSeries get Labels
func PromSeries(endpoint string, query string, vars []Var, start time.Time, end time.Time) (string, error) {
	client, err := api.NewClient(api.Config{
		Address: endpoint,
	})
	if err != nil {
		log.Error("Error creating client:", err)
		return "nil", err
	}

	v1api := v1.NewAPI(client)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	lbls, warnings, err := v1api.Series(ctx, []string{
		query,
	}, start, end)
	if err != nil {
		log.Error("Error querying Prometheus:", err)
		// os.Exit(1)
		return "nil", err
	}
	if len(warnings) > 0 {
		log.Warn("Warnings:", warnings)
	}
	var sb strings.Builder
	for _, lbl := range lbls {
		sb.WriteString(lbl.String())
		sb.WriteString(" ")
	}
	result := sb.String()
	log.Trace("Result:", result)
	return result, nil
}

// SubstVars replace vars in query
func SubstVars(query string, vars []Var) (string, error) {
	//add Global Vars to Vars
	for _, variable := range vars {
		query = strings.Replace(query, "$"+variable.Name, variable.Value, -1)
	}
	//no $ should be in query now (except $1... used in label_replace promQL)
	matched, _ := regexp.MatchString("\\$[a-z].*", query)
	if matched {
		log.Error("Missing a var for query ", query)
		return query, errors.New("Missing a var for query for query " + query)
	}
	return query, nil
}
