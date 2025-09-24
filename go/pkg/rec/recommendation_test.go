package rec

import (
	"testing"
	"vpr/pkg/utils"
)

func TestFindLimitAlias3_MatchFound(t *testing.T) {
	r := &Recommender{
		ExtraParams: []utils.PodContainerExtraParams{
			{Pod: "tmp-(coucou)-\\d", Container: "tmp-container", LimitAlias: "res.tmp"},
			{Pod: "other-pod", Container: "other-container", LimitAlias: "alias2"},
		},
	}
	podGroup := PodGroup{Name: "tmp-coucou-1"}
	containerName := "tmp-container"

	got, _, _, _ := r.findExtraParams(podGroup, containerName)
	want := "res.tmp"
	if got != want {
		t.Errorf("findExtraParams() = %v, want %v", got, want)
	}
}

func TestFindLimitAlias2_MatchFound(t *testing.T) {
	r := &Recommender{
		ExtraParams: []utils.PodContainerExtraParams{
			{Pod: "tmp-editor(-.*)", Container: "tmp-editor", LimitAlias: "res.editor"},
			{Pod: "other-pod", Container: "other-container", LimitAlias: "alias2"},
		},
	}
	podGroup := PodGroup{Name: "tmp-editor-slave-1"}
	containerName := "tmp-editor"

	got, _, _, _ := r.findExtraParams(podGroup, containerName)
	want := "res.editor"
	if got != want {
		t.Errorf("findExtraParams() = %v, want %v", got, want)
	}
}

func TestFindLimitAlias_MatchFound(t *testing.T) {
	r := &Recommender{
		ExtraParams: []utils.PodContainerExtraParams{
			{Pod: "my-pod", Container: "my-container", LimitAlias: "alias1"},
			{Pod: "other-pod", Container: "other-container", LimitAlias: "alias2"},
		},
	}
	podGroup := PodGroup{Name: "my-pod"}
	containerName := "my-container"

	got, _, _, _ := r.findExtraParams(podGroup, containerName)
	want := "alias1"
	if got != want {
		t.Errorf("findExtraParams() = %v, want %v", got, want)
	}
}

func TestFindLimitAlias_MatchWithRegex(t *testing.T) {
	r := &Recommender{
		ExtraParams: []utils.PodContainerExtraParams{
			{Pod: "my-.*", Container: "container-\\d+", LimitAlias: "regex-alias"},
		},
	}
	podGroup := PodGroup{Name: "my-pod"}
	containerName := "container-123"

	got, _, _, _ := r.findExtraParams(podGroup, containerName)
	want := "regex-alias"
	if got != want {
		t.Errorf("findExtraParams() = %v, want %v", got, want)
	}
}

func TestFindLimitAlias_NoMatch(t *testing.T) {
	r := &Recommender{
		ExtraParams: []utils.PodContainerExtraParams{
			{Pod: "foo", Container: "bar", LimitAlias: "alias"},
		},
	}
	podGroup := PodGroup{Name: "baz"}
	containerName := "qux"

	got, _, _, _ := r.findExtraParams(podGroup, containerName)
	want := "NA"
	if got != want {
		t.Errorf("findExtraParams() = %v, want %v", got, want)
	}
}

func TestFindLimitAlias_EmptyAliases(t *testing.T) {
	r := &Recommender{
		ExtraParams: []utils.PodContainerExtraParams{},
	}
	podGroup := PodGroup{Name: "any"}
	containerName := "any"

	got, _, _, _ := r.findExtraParams(podGroup, containerName)
	want := "NA"
	if got != want {
		t.Errorf("findExtraParams() = %v, want %v", got, want)
	}
}

func TestReplaceCaptureGroup(t *testing.T) {
	tests := []struct {
		pattern     string
		str         string
		replacement string
		expected    string
	}{
		{"tmp-server-(.*)", "tmp-server-site-1", "nx-$1", "nx-site-1"},
		{"my-pod-(.*)", "my-pod-xyz", "alias-$1", "alias-xyz"},
		{"no-capture", "no-capture", "alias-$1", "alias-$1"}, // No capture group, should remain unchanged
	}

	for _, tt := range tests {
		t.Run(tt.str, func(t *testing.T) {
			result := replaceCaptureGroup(tt.pattern, tt.str, tt.replacement)
			if result != tt.expected {
				t.Errorf("replaceCaptureGroup(%q, %q, %q) = %q; want %q", tt.pattern, tt.str, tt.replacement, result, tt.expected)
			}
		})
	}
}
