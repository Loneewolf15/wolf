package packager

import (
	"testing"
)

func TestSatisfies(t *testing.T) {
	tests := []struct {
		version    string
		constraint string
		expected   bool
	}{
		{"1.2.3", "^1.2.0", true},
		{"1.3.0", "^1.2.0", true},
		{"2.0.0", "^1.2.0", false},
		{"1.2.3", "~1.2.0", true},
		{"1.3.0", "~1.2.0", false},
		{"2.0.0", ">=1.0.0", true},
		{"0.9.0", ">=1.0.0", false},
		{"v1.2.3", "^1.2.0", true},
		{"1.0.0-rc.1", "latest", false},
		{"1.0.0", "latest", true},
		{"1.2.3", "1.2.3", true},
		{"1.2.4", "1.2.3", false},
		{"0.1.5", "^0.1.4", true},
		{"0.2.0", "^0.1.4", false},
	}

	for _, test := range tests {
		result := Satisfies(test.version, test.constraint)
		if result != test.expected {
			t.Errorf("Satisfies(%q, %q) = %v; want %v", test.version, test.constraint, result, test.expected)
		}
	}
}

func TestHighestCompatible(t *testing.T) {
	tags := []string{"v1.0.0", "v1.1.0", "v1.2.0", "v2.0.0", "v1.2.1-beta", "main"}

	tests := []struct {
		constraint string
		expected   string
	}{
		{"^1.0.0", "v1.2.0"},
		{"~1.1.0", "v1.1.0"},
		{">=1.0.0", "v2.0.0"},
		{"v1.2.1-beta", "v1.2.1-beta"},
		{"latest", "v2.0.0"},
		{"main", "main"}, // fallback to branch name
	}

	for _, test := range tests {
		result := HighestCompatible(test.constraint, tags)
		if result != test.expected {
			t.Errorf("HighestCompatible(%q, tags) = %q; want %q", test.constraint, result, test.expected)
		}
	}
}
