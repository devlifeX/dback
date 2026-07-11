package shell

import "testing"

func TestExecutionPlanHasSeparateSteps(t *testing.T) {
	plan := ExecutionPlan{
		Mode: ModeRemoteToLocalPipe,
		Steps: []Command{
			{Binary: "tar", Args: []string{"-cf", "-", "-C", "/var/www", "--", "html"}},
			{Binary: "zstd", Args: []string{"-1"}},
		},
	}
	if len(plan.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(plan.Steps))
	}
	if plan.Steps[0].Binary != "tar" || plan.Steps[1].Binary != "zstd" {
		t.Fatalf("unexpected steps: %#v", plan.Steps)
	}
}

func TestExecutionPlanStepsSerializeWithoutPipeline(t *testing.T) {
	plan := ExecutionPlan{
		Mode: ModeLocalPipe,
		Steps: []Command{
			{Binary: "tar", Args: []string{"-cf", "-"}},
			{Binary: "gzip", Args: []string{"-1"}},
		},
	}
	for _, step := range plan.Steps {
		s, err := SerializePOSIX(step)
		if err != nil {
			t.Fatal(err)
		}
		if s == "" {
			t.Fatal("expected serialized command")
		}
	}
}
