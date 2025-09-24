package rec

import (
	"testing"
)

func TestGetMaxAfterFullGC(t *testing.T) {
	tests := []struct {
		name     string
		values   []float64
		expected float64
	}{
		{
			name:     "Single peak with no GC",
			values:   []float64{1, 2, 3, 4, 5},
			expected: -1.0,
		},
		{
			name:     "Single GC with max after GC",
			values:   []float64{1, 2, 3, 2, 4},
			expected: 2.0,
		},
		{
			name:     "Multiple GCs with max after last GC",
			values:   []float64{1, 3, 2, 4, 1, 5},
			expected: 2.0,
		},
		{
			name:     "Multiple GCs with max after first GC",
			values:   []float64{1, 3, 2, 5, 1, 4},
			expected: 2.0,
		},
		{
			name:     "No values",
			values:   []float64{},
			expected: -1.0,
		},
		{
			name:     "All values are the same",
			values:   []float64{2, 2, 2, 2},
			expected: -1.0,
		},
		{
			name:     "Decreasing values",
			values:   []float64{5, 4, 3, 2, 1},
			expected: -1.0,
			// expected: 4.0,
		},
		{
			name:     "Single value",
			values:   []float64{10},
			expected: -1.0,
		},
		{
			name:     "new 2 times down",
			values:   []float64{262, 332, 794, 620, 158, 196, 274},
			expected: 158.0,
		},
		{
			name:     "new 3 min 2nd one wins",
			values:   []float64{5, 4, 3, 2, 1, 2, 3, 4, 5, 4, 3, 4, 5, 4, 3, 2, 3},
			expected: 3.0,
		},
		{
			name:     "new 3 min avoid zero 3rd one wins",
			values:   []float64{5, 4, 3, 2, 1, 2, 3, 4, 5, 4, 0, 4, 5, 4, 3, 2, 3},
			expected: 2.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getMaxAfterFullGC(tt.values)
			if result != tt.expected {
				t.Errorf("getMaxAfterFullGC(%v) = %v; want %v", tt.values, result, tt.expected)
			}
		})
	}
}
