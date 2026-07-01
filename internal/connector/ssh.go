package connector

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"

	"dback/backend/shell"
	"dback/backend/ssh"
	"dback/models"
)

type sshConnector struct {
	client ssh.Executor
}

func newSSHConnector(profile models.Profile) (Connector, error) {
	client, err := ssh.NewExecutor(profile)
	if err != nil {
		return nil, err
	}
	return &sshConnector{client: client}, nil
}

func (c *sshConnector) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

func (c *sshConnector) Run(ctx context.Context, plan shell.ExecutionPlan) (*shell.StreamResult, error) {
	if len(plan.Steps) == 0 {
		return nil, fmt.Errorf("execution plan has no steps")
	}
	switch plan.Mode {
	case shell.ModeRemoteToLocalPipe:
		return c.runRemoteToLocalPipe(ctx, plan.Steps)
	default:
		return nil, fmt.Errorf("ssh connector requires remote-to-local pipe mode")
	}
}

func (c *sshConnector) runRemoteToLocalPipe(ctx context.Context, steps []shell.Command) (*shell.StreamResult, error) {
	if len(steps) < 2 {
		return nil, fmt.Errorf("remote-to-local pipe requires at least two steps")
	}
	return c.runRemotePipeline(ctx, steps)
}

func (c *sshConnector) runRemotePipeline(ctx context.Context, steps []shell.Command) (*shell.StreamResult, error) {
	firstCmd, err := shell.SerializePOSIX(steps[0])
	if err != nil {
		return nil, err
	}
	currentOut, firstErr, currentSession, err := c.client.RunCommandStream(firstCmd)
	if err != nil {
		return nil, err
	}

	var stderrReaders []io.Reader
	stderrReaders = append(stderrReaders, firstErr)
	var sessions []ssh.Session
	sessions = append(sessions, currentSession)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstPipelineErr error
	setErr := func(err error) {
		if err == nil {
			return
		}
		mu.Lock()
		if firstPipelineErr == nil {
			firstPipelineErr = err
		}
		mu.Unlock()
	}

	for i := 1; i < len(steps); i++ {
		cmdString, err := shell.SerializePOSIX(steps[i])
		if err != nil {
			for _, session := range sessions {
				_ = session.Close()
			}
			return nil, err
		}
		stdin, stdout, stderr, nextSession, err := c.client.RunCommandPipe(cmdString)
		if err != nil {
			for _, session := range sessions {
				_ = session.Close()
			}
			return nil, err
		}
		prevOut := currentOut
		prevSession := currentSession
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer stdin.Close()
			_, err := io.Copy(stdin, prevOut)
			setErr(err)
			setErr(prevSession.Wait())
		}()
		currentOut = stdout
		currentSession = nextSession
		sessions = append(sessions, nextSession)
		stderrReaders = append(stderrReaders, stderr)
	}

	go func() {
		<-ctx.Done()
		for _, session := range sessions {
			_ = session.Close()
		}
	}()

	wait := onceWait(func() error {
		wg.Wait()
		mu.Lock()
		err := firstPipelineErr
		mu.Unlock()
		if err != nil {
			return err
		}
		return currentSession.Wait()
	})

	return &shell.StreamResult{
		Reader: readCloser{Reader: currentOut, closeFn: func() error {
			return wait()
		}},
		Stderr: io.MultiReader(stderrReaders...),
		Wait:   wait,
	}, nil
}

func startLocalPipeline(ctx context.Context, steps []shell.Command, in io.Reader) (io.ReadCloser, io.Reader, func() error, error) {
	if len(steps) == 1 {
		cmd := exec.CommandContext(ctx, steps[0].Binary, steps[0].Args...)
		cmd.Stdin = in
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return nil, nil, nil, err
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return nil, nil, nil, err
		}
		if err := cmd.Start(); err != nil {
			return nil, nil, nil, err
		}
		wait := onceWait(func() error { return cmd.Wait() })
		return readCloser{Reader: stdout, closeFn: wait}, stderr, wait, nil
	}

	midReader, midWriter := io.Pipe()
	midOut, midErr, midWait, err := startLocalPipeline(ctx, steps[1:], midReader)
	if err != nil {
		return nil, nil, nil, err
	}

	cmd := exec.CommandContext(ctx, steps[0].Binary, steps[0].Args...)
	cmd.Stdin = in
	cmd.Stdout = midWriter
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, nil, err
	}

	var wg sync.WaitGroup
	var stepErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer midWriter.Close()
		if err := cmd.Wait(); err != nil && stepErr == nil {
			stepErr = err
		}
	}()

	wait := onceWait(func() error {
		wg.Wait()
		if stepErr != nil {
			return stepErr
		}
		return midWait()
	})

	return midOut, io.MultiReader(stderr, midErr), wait, nil
}
