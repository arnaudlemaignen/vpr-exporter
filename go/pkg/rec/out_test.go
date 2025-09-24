package rec

import (
	"fmt"
	"regexp"
	"testing"
)

func TestGenDimValues(t *testing.T) {
	// Define test cases
	tests := []struct {
		name     string
		input    []Recommendation
		expected string
	}{
		{
			name: "Single Recommendation with Namespace tmp",
			input: []Recommendation{
				{
					Namespace:     "tmp",
					PodGroupName:  "pod-group-1",
					ContainerName: "container-1",
					NewCPUReqM:    500,
					NewMemReqMB:   256,
					NewMemLimitMB: 512,
					GainCPUReqM:   51,
					GainMemReqMB:  101,
					LimitAlias:    "res.pod_group_1.container_1",
				},
			},
			expected: `# VPR recommendations
res:
  pod_group_1:
    container_1:
      # tmp | pod-group-1 | container-1
      requests:
        cpu: 500m # Gain 51m
        memory: 256Mi # Gain 101 Mi
      limits:
        memory: 512Mi
# Overall gain on CPU req 51 m | Mem req 101 Mi
`,
		},
		{
			name: "Multiple Recommendations with mixed namespaces",
			input: []Recommendation{
				{
					Namespace:     "tmp",
					PodGroupName:  "pod-group-1",
					ContainerName: "container-1",
					NewCPUReqM:    500,
					NewMemReqMB:   256,
					NewMemLimitMB: 512,
					GainCPUReqM:   51,
					GainMemReqMB:  101,
					LimitAlias:    "res.pod_group_1.container_1",
				},
				{
					Namespace:     "other",
					PodGroupName:  "pod-group-2",
					ContainerName: "container-2",
					NewCPUReqM:    1000,
					NewMemReqMB:   512,
					NewMemLimitMB: 1024,
				},
			},
			expected: `# VPR recommendations
res:
  pod_group_1:
    container_1:
      # tmp | pod-group-1 | container-1
      requests:
        cpu: 500m # Gain 51m
        memory: 256Mi # Gain 101 Mi
      limits:
        memory: 512Mi
# Overall gain on CPU req 51 m | Mem req 101 Mi
`,
		},
		{
			name:     "Empty Recommendations",
			input:    []Recommendation{},
			expected: "# VPR recommendations\n# Overall gain on CPU req 0 m | Mem req 0 Mi\n",
		},
	}

	// Create a recommender instance
	r := NewRecommender(nil) // Assuming nil for limit aliases, as they are not relevant for this test
	// Run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.genDimValues(tt.input)
			if result != tt.expected {
				t.Errorf("genDimValues() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestReplaceDashByUnderscore(t *testing.T) {
	// Define test cases
	tests := []struct {
		input    string
		expected string
	}{
		{"pod-group-1", "pod_group_1"},
		{"container-name", "container_name"},
		{"no-dashes", "no_dashes"},
		{"", ""},
	}

	// Run test cases
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := replaceDashByUnderscore(tt.input)
			if result != tt.expected {
				t.Errorf("replaceDashByUnderscore() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestRegex(t *testing.T) {

	test := "^tmp-gateway-(coucou)-\\d$"
	matchContainer, _ := regexp.MatchString(test, "tmp-gateway-coucou-1")
	fmt.Println(matchContainer)
	matchContainer, _ = regexp.MatchString(test, "tmp-gateway-coucou")
	fmt.Println(matchContainer)

	test = "^tmp-tmp-gateway-editor(-.*)$"
	matchContainer, _ = regexp.MatchString(test, "tmp-tmp-gateway-editor-coucou-1")
	fmt.Println(matchContainer)
}
