package connector

import (
	"context"
	"io"
	"strings"
	"testing"

	"dback/backend/shell"
	"dback/backend/ssh"
)

type failingWaitSession struct{}

func (failingWaitSession) Close() error { return nil }
func (failingWaitSession) Wait() error  { return io.ErrUnexpectedEOF }

func TestSSHConnectorPropagatesWaitError(t *testing.T) {
	c := &sshConnector{client: &waitFailSSHExecutor{}}
	result, err := c.Run(context.Background(), shell.ExecutionPlan{
		Mode: shell.ModeRemoteToLocalPipe,
		Steps: []shell.Command{
			{Binary: "tar", Args: []string{"-cf", "-"}},
			{Binary: "zstd", Args: []string{"-1"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadAll(result.Reader); err != nil {
		t.Fatal(err)
	}
	if err := result.Wait(); err == nil {
		t.Fatal("expected wait error")
	}
}

type waitFailSSHExecutor struct{}

func (waitFailSSHExecutor) RunCommandStream(cmd string) (io.Reader, io.Reader, ssh.Session, error) {
	return strings.NewReader("data"), strings.NewReader(""), failingWaitSession{}, nil
}

func (waitFailSSHExecutor) RunCommandPipeInput(cmd string) (io.WriteCloser, io.Reader, ssh.Session, error) {
	return nil, nil, nil, nil
}

func (waitFailSSHExecutor) RunCommandPipe(cmd string) (io.WriteCloser, io.Reader, io.Reader, ssh.Session, error) {
	inReader, inWriter := io.Pipe()
	outReader, outWriter := io.Pipe()
	go func() {
		defer inReader.Close()
		defer outWriter.Close()
		_, _ = io.Copy(outWriter, inReader)
	}()
	return inWriter, outReader, strings.NewReader(""), failingWaitSession{}, nil
}

func (waitFailSSHExecutor) RunCommand(cmd string) (string, error) { return "", nil }
func (waitFailSSHExecutor) Close() error                          { return nil }
