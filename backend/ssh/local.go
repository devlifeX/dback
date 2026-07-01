package ssh

import (
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"
)

// localCmd returns an *exec.Cmd that executes cmd via bash.
// On Windows, wsl.exe is used so the same POSIX command strings work.
func localCmd(cmd string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("wsl.exe", "bash", "-c", cmd)
	}
	return exec.Command("bash", "-c", cmd)
}

// LocalClient runs database shell commands on the local machine (no SSH).
type LocalClient struct{}

func (c *LocalClient) Close() error {
	return nil
}

func (c *LocalClient) RunCommand(cmd string) (string, error) {
	out, err := localCmd(cmd).CombinedOutput()
	return string(out), err
}

func (c *LocalClient) RunCommandStream(cmd string) (io.Reader, io.Reader, Session, error) {
	command := localCmd(cmd)
	stdout, err := command.StdoutPipe()
	if err != nil {
		return nil, nil, nil, err
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		return nil, nil, nil, err
	}
	if err := command.Start(); err != nil {
		return nil, nil, nil, err
	}
	return stdout, stderr, &localSession{cmd: command}, nil
}

func (c *LocalClient) RunCommandPipeInput(cmd string) (io.WriteCloser, io.Reader, Session, error) {
	command := localCmd(cmd)
	stdin, err := command.StdinPipe()
	if err != nil {
		return nil, nil, nil, err
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		return nil, nil, nil, err
	}
	if err := command.Start(); err != nil {
		return nil, nil, nil, err
	}
	return stdin, stderr, &localSession{cmd: command}, nil
}

func (c *LocalClient) RunCommandPipe(cmd string) (io.WriteCloser, io.Reader, io.Reader, Session, error) {
	command := localCmd(cmd)
	stdin, err := command.StdinPipe()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if err := command.Start(); err != nil {
		return nil, nil, nil, nil, err
	}
	return stdin, stdout, stderr, &localSession{cmd: command}, nil
}

type localSession struct {
	cmd *exec.Cmd
}

func (s *localSession) Close() error {
	if s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
	return nil
}

func (s *localSession) Wait() error {
	err := s.cmd.Wait()
	if err == nil {
		return nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		msg := strings.TrimSpace(string(exitErr.Stderr))
		if msg != "" {
			return fmt.Errorf("%w: %s", err, msg)
		}
	}
	return err
}
