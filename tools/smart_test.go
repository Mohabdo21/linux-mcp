package tools

import "testing"

func TestGatherIOStatsMissingBinaryIsNotFatal(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	out, err := GatherIOStats(t.Context())
	if err != nil {
		t.Fatalf(
			"GatherIOStats() error = %v, want nil: a missing iostat is not fatal",
			err,
		)
	}
	if out == nil {
		t.Fatal("GatherIOStats() = nil, want non-nil output carrying the error")
	}
	if out.Devices == nil {
		t.Error(
			"GatherIOStats() Devices = nil, want an empty slice so it marshals as [] not null",
		)
	}
	if out.ErrorCount() == 0 {
		t.Error(
			"GatherIOStats() recorded no error, want iostat reported in errors",
		)
	}
}

func TestGatherSMARTHealthMissingBinaryIsNotFatal(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	out, err := GatherSMARTHealth(t.Context(), "/dev/sda")
	if err != nil {
		t.Fatalf(
			"GatherSMARTHealth() error = %v, want nil: a missing smartctl is not fatal",
			err,
		)
	}
	if out == nil {
		t.Fatal(
			"GatherSMARTHealth() = nil, want non-nil output carrying the error",
		)
	}
	if out.Devices == nil {
		t.Error(
			"GatherSMARTHealth() Devices = nil, want an empty slice so it marshals as [] not null",
		)
	}
	if out.ErrorCount() == 0 {
		t.Error(
			"GatherSMARTHealth() recorded no error, want smartctl reported in errors",
		)
	}
}
