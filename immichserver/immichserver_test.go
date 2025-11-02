package immichserver

import "testing"

func TestMinimumVersion(t *testing.T) {
	table := []struct {
		l        ImmichServerVersion
		r        ImmichServerVersion
		expected bool
	}{
		{ImmichServerVersion{1, 0, 0}, ImmichServerVersion{0, 0, 0}, true},
		{ImmichServerVersion{1, 1, 0}, ImmichServerVersion{1, 1, 0}, true},
		{ImmichServerVersion{1, 0, 0}, ImmichServerVersion{1, 0, 0}, true},
		{ImmichServerVersion{1, 0, 0}, ImmichServerVersion{1, 1, 0}, false},
		{ImmichServerVersion{1, 1, 0}, ImmichServerVersion{1, 1, 0}, true},
		{ImmichServerVersion{1, 1, 0}, ImmichServerVersion{1, 1, 1}, false},
		{ImmichServerVersion{2, 0, 0}, ImmichServerVersion{1, 1, 1}, true},
		{ImmichServerVersion{1, 1, 0}, ImmichServerVersion{1, 0, 2}, true},
		{ImmichServerVersion{1, 2, 0}, ImmichServerVersion{2, 0, 0}, true},
	}
	for _, scenario := range table {
		if scenario.l.IsMinimumVersion(scenario.r) != scenario.expected {
			t.Errorf("Expected '%v >= %v' to be %v, but it was not", scenario.l, scenario.r, scenario.expected)
		}
	}
}
