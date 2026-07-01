package shell

import (
	"strings"
	"testing"
)

func TestSerializePOSIXQuotesArgs(t *testing.T) {
	cmd := Command{
		Binary: "tar",
		Args:   []string{"-cf", "-", "--exclude", "it's", "/var/www/html"},
	}
	s, err := SerializePOSIX(cmd)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(s, "tar ... |") {
		t.Fatal("must not contain pipeline")
	}
	if !strings.Contains(s, "it'\\''s") && !strings.Contains(s, `'it's'`) {
		t.Fatalf("expected quoted arg, got %q", s)
	}
}

func TestExecutionPlanNoPipeString(t *testing.T) {
	plan := ExecutionPlan{
		Mode: ModeRemoteToLocalPipe,
		Steps: []Command{
			{Binary: "tar", Args: []string{"-cf", "-"}},
			{Binary: "zstd", Args: []string{"-1"}},
		},
	}
	if len(plan.Steps) != 2 {
		t.Fatalf("expected 2 steps")
	}
	for _, step := range plan.Steps {
		if strings.Contains(strings.Join(step.Args, " "), "|") {
			t.Fatal("step args must not include pipe")
		}
	}
}
