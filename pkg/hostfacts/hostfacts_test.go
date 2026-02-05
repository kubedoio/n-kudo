package hostfacts

import "testing"

func TestCollect(t *testing.T) {
	facts, err := Collect()
	if err != nil {
		t.Fatalf("collect failed: %v", err)
	}
	if facts.CPUCores <= 0 {
		t.Fatalf("expected cpu cores > 0, got %d", facts.CPUCores)
	}
	if facts.Arch == "" {
		t.Fatalf("expected arch")
	}
	if len(facts.Disks) == 0 {
		t.Fatalf("expected at least one disk entry")
	}
}
