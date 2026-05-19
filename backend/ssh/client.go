package ssh

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"time"

	"dback/models"

	"golang.org/x/crypto/ssh"
)

// Client wraps the ssh.Client and provides high-level operations
type Client struct {
	conn     *ssh.Client
	jumpConn *ssh.Client
}

// NewClient creates a new SSH client based on the profile
func NewClient(p models.Profile) (*Client, error) {
	targetConfig, err := sshConfig(p.SSHUser, p.SSHPassword, p.AuthType, p.AuthKeyPath)
	if err != nil {
		return nil, err
	}

	targetAddr := net.JoinHostPort(p.Host, p.Port)
	if p.ConnectionType == models.ConnectionTypeJumpHost {
		return newJumpClient(p, targetConfig, targetAddr)
	}

	client, err := ssh.Dial("tcp", targetAddr, targetConfig)
	if err != nil {
		return nil, err
	}

	return &Client{conn: client}, nil
}

func sshConfig(user, password string, authType models.AuthType, keyPath string) (*ssh.ClientConfig, error) {
	config := &ssh.ClientConfig{
		User:            user,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // For simplicity; in prod, verify host keys
		Timeout:         10 * time.Second,
	}

	if authType == models.AuthTypePassword {
		config.Auth = []ssh.AuthMethod{
			ssh.Password(password),
		}
	} else {
		key, err := ioutil.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("unable to read private key: %v", err)
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("unable to parse private key: %v", err)
		}
		config.Auth = []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		}
	}

	return config, nil
}

func newJumpClient(p models.Profile, targetConfig *ssh.ClientConfig, targetAddr string) (*Client, error) {
	jumpPort := p.JumpPort
	if jumpPort == "" {
		jumpPort = "22"
	}
	jumpAuthType := p.JumpAuthType
	if jumpAuthType == "" {
		jumpAuthType = models.AuthTypePassword
	}
	jumpConfig, err := sshConfig(p.JumpUser, p.JumpPassword, jumpAuthType, p.JumpAuthKeyPath)
	if err != nil {
		return nil, fmt.Errorf("jump host auth failed: %w", err)
	}
	jumpAddr := net.JoinHostPort(p.JumpHost, jumpPort)
	jumpClient, err := ssh.Dial("tcp", jumpAddr, jumpConfig)
	if err != nil {
		return nil, fmt.Errorf("jump host connection failed: %w", err)
	}

	targetConn, err := jumpClient.Dial("tcp", targetAddr)
	if err != nil {
		jumpClient.Close()
		return nil, fmt.Errorf("target connection through jump host failed: %w", err)
	}

	conn, chans, reqs, err := ssh.NewClientConn(targetConn, targetAddr, targetConfig)
	if err != nil {
		targetConn.Close()
		jumpClient.Close()
		return nil, fmt.Errorf("target ssh handshake through jump host failed: %w", err)
	}

	return &Client{
		conn:     ssh.NewClient(conn, chans, reqs),
		jumpConn: jumpClient,
	}, nil
}

// Close closes the SSH connection
func (c *Client) Close() error {
	if c.conn != nil {
		err := c.conn.Close()
		if c.jumpConn != nil {
			if jumpErr := c.jumpConn.Close(); err == nil {
				err = jumpErr
			}
		}
		return err
	}
	if c.jumpConn != nil {
		return c.jumpConn.Close()
	}
	return nil
}

// RunCommandStream executes a command and returns its stdout pipe and stderr pipe.
// This is crucial for streaming large dumps.
func (c *Client) RunCommandStream(cmd string) (io.Reader, io.Reader, *ssh.Session, error) {
	session, err := c.conn.NewSession()
	if err != nil {
		return nil, nil, nil, err
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		return nil, nil, nil, err
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		session.Close()
		return nil, nil, nil, err
	}

	if err := session.Start(cmd); err != nil {
		session.Close()
		return nil, nil, nil, err
	}

	return stdout, stderr, session, nil
}

// RunCommandPipeInput executes a command and returns its stdin pipe and stderr pipe.
// This is used for uploading/restoring dumps.
func (c *Client) RunCommandPipeInput(cmd string) (io.WriteCloser, io.Reader, *ssh.Session, error) {
	session, err := c.conn.NewSession()
	if err != nil {
		return nil, nil, nil, err
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		session.Close()
		return nil, nil, nil, err
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		session.Close()
		return nil, nil, nil, err
	}

	if err := session.Start(cmd); err != nil {
		session.Close()
		return nil, nil, nil, err
	}

	return stdin, stderr, session, nil
}

// RunCommand executes a command and returns combined stdout/stderr
func (c *Client) RunCommand(cmd string) (string, error) {
	session, err := c.conn.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	return string(output), err
}

// ProgressReader wraps an io.Reader to report progress
type ProgressReader struct {
	Reader   io.Reader
	Total    int64
	Current  int64
	Callback func(current int64, total int64)
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.Current += int64(n)
	if pr.Callback != nil {
		pr.Callback(pr.Current, pr.Total)
	}
	return n, err
}
