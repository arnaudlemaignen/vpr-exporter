package utils

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubstVars(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		vars      []Var
		expected  string
		expectErr error
	}{
		{
			name:     "All variables substituted",
			query:    "rate($metric[5m])",
			vars:     []Var{{Name: "metric", Value: "http_requests_total"}},
			expected: "rate(http_requests_total[5m])",
		},
		{
			name:      "Missing variable substitution",
			query:     "rate($metric[5m])",
			vars:      []Var{},
			expected:  "rate($metric[5m])",
			expectErr: errors.New("Missing a var for query for query rate($metric[5m])"),
		},
		{
			name:     "No variables in query",
			query:    "rate(http_requests_total[5m])",
			vars:     []Var{},
			expected: "rate(http_requests_total[5m])",
		},
		{
			name:     "Multiple variables substituted",
			query:    "rate($metric[5m]) + $value",
			vars:     []Var{{Name: "metric", Value: "http_requests_total"}, {Name: "value", Value: "5"}},
			expected: "rate(http_requests_total[5m]) + 5",
		},
		{
			name:     "Variable with special characters",
			query:    "rate($metric[5m])",
			vars:     []Var{{Name: "metric", Value: "http_requests_total{job=\"api\"}"}},
			expected: "rate(http_requests_total{job=\"api\"}[5m])",
		},
		{
			name:     "Variable with typical podgroup",
			query:    `max by(statefulset,namespace)(kube_statefulset_status_replicas{namespace=~"$namespace"}) > 0`,
			vars:     []Var{{Name: "namespace", Value: "temp"}},
			expected: "max by(statefulset,namespace)(kube_statefulset_status_replicas{namespace=~\"temp\"}) > 0",
		},
		{
			name:     "Variable with typical podgroup",
			query:    `max by(statefulset,namespace)(kube_statefulset_status_replicas{namespace=~"$namespace"}) > 0`,
			vars:     []Var{{Name: "namespace", Value: ".*"}},
			expected: "max by(statefulset,namespace)(kube_statefulset_status_replicas{namespace=~\".*\"}) > 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SubstVars(tt.query, tt.vars)
			if tt.expectErr != nil {
				assert.EqualError(t, err, tt.expectErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}
