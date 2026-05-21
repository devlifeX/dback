package ssh

import (
	"io"
	"strings"
	"testing"

	"dback/models"
)

func TestNewExecutorLocalhost(t *testing.T) {
	client, err := NewExecutor(models.Profile{ConnectionType: models.ConnectionTypeLocalhost})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if _, ok := client.(*LocalClient); !ok {
		t.Fatalf("expected LocalClient, got %T", client)
	}
}

func TestLocalClientRunCommand(t *testing.T) {
	client := &LocalClient{}
	out, err := client.RunCommand("echo dback-localhost-ok")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "dback-localhost-ok") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestLocalClientRunCommandStream(t *testing.T) {
	client := &LocalClient{}
	stdout, stderr, session, err := client.RunCommandStream("echo streamed")
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	data, err := io.ReadAll(stdout)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "streamed") {
		t.Fatalf("unexpected stdout: %q", data)
	}
	_, _ = io.ReadAll(stderr)
	if err := session.Wait(); err != nil {
		t.Fatal(err)
	}
}

func TestNewClientRejectsLocalhost(t *testing.T) {
	_, err := NewClient(models.Profile{ConnectionType: models.ConnectionTypeLocalhost})
	if err == nil {
		t.Fatal("expected error for localhost profile")
	}
}
