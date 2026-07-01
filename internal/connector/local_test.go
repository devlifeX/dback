package connector

import (
	"context"
	"io"
	"strings"
	"testing"

	"dback/backend/shell"
)

func TestLocalConnectorPipesStepsAndWaitsOnce(t *testing.T) {
	c := newLocalConnector()
	result, err := c.Run(context.Background(), shell.ExecutionPlan{
		Mode: shell.ModeLocalPipe,
		Steps: []shell.Command{
			{Binary: "printf", Args: []string{"hello"}},
			{Binary: "tr", Args: []string{"a-z", "A-Z"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(result.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(out); got != "HELLO" {
		t.Fatalf("expected piped output HELLO, got %q", got)
	}
	if err := result.Wait(); err != nil {
		t.Fatal(err)
	}
	if err := result.Wait(); err != nil {
		t.Fatalf("wait should be idempotent: %v", err)
	}
}

func TestLocalConnectorPropagatesWaitError(t *testing.T) {
	c := newLocalConnector()
	result, err := c.Run(context.Background(), shell.ExecutionPlan{
		Mode: shell.ModeLocalPipe,
		Steps: []shell.Command{
			{Binary: "sh", Args: []string{"-c", "printf partial; exit 7"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(result.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(out)) != "partial" {
		t.Fatalf("unexpected output: %q", string(out))
	}
	if err := result.Wait(); err == nil {
		t.Fatal("expected wait error")
	}
}
