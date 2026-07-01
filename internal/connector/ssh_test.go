package connector

import (
	"context"
	"io"
	"strings"
	"sync"
	"testing"

	"dback/backend/shell"
	"dback/backend/ssh"
)

type fakeSSHExecutor struct {
	streamCmds []string
	pipeCmds   []string
}

func (f *fakeSSHExecutor) RunCommandStream(cmd string) (io.Reader, io.Reader, ssh.Session, error) {
	f.streamCmds = append(f.streamCmds, cmd)
	return strings.NewReader("hello"), strings.NewReader(""), fakeSession{}, nil
}

func (f *fakeSSHExecutor) RunCommandPipeInput(cmd string) (io.WriteCloser, io.Reader, ssh.Session, error) {
	return nil, nil, nil, nil
}

func (f *fakeSSHExecutor) RunCommandPipe(cmd string) (io.WriteCloser, io.Reader, io.Reader, ssh.Session, error) {
	f.pipeCmds = append(f.pipeCmds, cmd)
	inReader, inWriter := io.Pipe()
	outReader, outWriter := io.Pipe()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer outWriter.Close()
		defer inReader.Close()
		data, _ := io.ReadAll(inReader)
		_, _ = outWriter.Write([]byte(strings.ToUpper(string(data))))
	}()
	return inWriter, outReader, strings.NewReader(""), fakeWaitSession{wait: wg.Wait}, nil
}

func (f *fakeSSHExecutor) RunCommand(cmd string) (string, error) { return "", nil }
func (f *fakeSSHExecutor) Close() error                          { return nil }

type fakeSession struct{}

func (fakeSession) Close() error { return nil }
func (fakeSession) Wait() error  { return nil }

type fakeWaitSession struct {
	wait func()
}

func (s fakeWaitSession) Close() error { return nil }
func (s fakeWaitSession) Wait() error {
	if s.wait != nil {
		s.wait()
	}
	return nil
}

func TestSSHConnectorRunsRemoteStepsWithoutPipelineString(t *testing.T) {
	fake := &fakeSSHExecutor{}
	c := &sshConnector{client: fake}
	result, err := c.Run(context.Background(), shell.ExecutionPlan{
		Mode: shell.ModeRemoteToLocalPipe,
		Steps: []shell.Command{
			{Binary: "tar", Args: []string{"-cf", "-", "-C", "/var/www", "--", "html"}},
			{Binary: "zstd", Args: []string{"-1"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(result.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := result.Wait(); err != nil {
		t.Fatal(err)
	}
	if string(out) != "HELLO" {
		t.Fatalf("expected remote pipeline output HELLO, got %q", out)
	}
	if len(fake.streamCmds) != 1 || len(fake.pipeCmds) != 1 {
		t.Fatalf("expected one stream command and one pipe command, got %#v / %#v", fake.streamCmds, fake.pipeCmds)
	}
	for _, cmd := range append(fake.streamCmds, fake.pipeCmds...) {
		if strings.Contains(cmd, "|") {
			t.Fatalf("command must not contain shell pipeline: %q", cmd)
		}
	}
}
