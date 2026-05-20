package ssh

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"dback/internal/debug"
	"dback/models"

	"golang.org/x/crypto/ssh"
)

const (
	maxDialAttempts = 3
	dialTimeout     = 30 * time.Second
	tcpKeepAlive    = 30 * time.Second
)

// Client wraps the ssh.Client and provides high-level operations
type Client struct {
	conn     *ssh.Client
	jumpConn *ssh.Client
}

// NewClient creates a new SSH client based on the profile
func NewClient(p models.Profile) (*Client, error) {
	targetConfig, err := sshConfig(p.SSHUser, p.SSHPassword, p.AuthType, p.AuthKeyPath, p.AuthKeyPEM)
	if err != nil {
		return nil, err
	}

	targetAddr := net.JoinHostPort(p.Host, p.Port)
	if p.ConnectionType == models.ConnectionTypeJumpHost {
		return newJumpClient(p, targetConfig, targetAddr)
	}

	client, err := dialSSH(targetAddr, targetConfig)
	if err != nil {
		return nil, err
	}

	return &Client{conn: client}, nil
}

func sshConfig(user, password string, authType models.AuthType, keyPath, keyPEM string) (*ssh.ClientConfig, error) {
	hostKeyCB, err := hostKeyCallback()
	if err != nil {
		return nil, err
	}
	config := &ssh.ClientConfig{
		User:            user,
		HostKeyCallback: hostKeyCB,
		Timeout:         dialTimeout,
	}

	if authType == models.AuthTypePassword {
		config.Auth = []ssh.AuthMethod{
			ssh.Password(password),
		}
	} else {
		key, err := loadPrivateKey(keyPEM, keyPath)
		if err != nil {
			return nil, err
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

func loadPrivateKey(keyPEM, keyPath string) ([]byte, error) {
	if keyPEM != "" {
		return []byte(keyPEM), nil
	}
	if keyPath == "" {
		return nil, fmt.Errorf("no private key provided")
	}
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read private key: %v", err)
	}
	return key, nil
}

func dialTCP(addr string) (net.Conn, error) {
	dialer := &net.Dialer{
		Timeout:   dialTimeout,
		KeepAlive: tcpKeepAlive,
	}
	return dialer.Dial("tcp", addr)
}

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	retryable := []string{
		"timeout",
		"connection refused",
		"connection reset",
		"broken pipe",
		"eof",
		"i/o timeout",
		"no route to host",
		"network is unreachable",
		"temporarily unavailable",
	}
	for _, needle := range retryable {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	return false
}

func dialSSH(addr string, config *ssh.ClientConfig) (*ssh.Client, error) {
	backoffs := []time.Duration{0, time.Second, 2 * time.Second}
	var lastErr error

	for attempt, wait := range backoffs {
		if wait > 0 {
			time.Sleep(wait)
			debug.Errorf("SSH dial retry %d/%d to %s: %v", attempt+1, maxDialAttempts, addr, lastErr)
		}

		conn, err := dialTCP(addr)
		if err != nil {
			lastErr = err
			if !isRetryableError(err) || attempt == len(backoffs)-1 {
				return nil, err
			}
			continue
		}

		sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
		if err != nil {
			_ = conn.Close()
			lastErr = err
			if !isRetryableError(err) || attempt == len(backoffs)-1 {
				return nil, err
			}
			continue
		}

		return ssh.NewClient(sshConn, chans, reqs), nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("ssh dial failed for %s", addr)
	}
	return nil, lastErr
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
	jumpConfig, err := sshConfig(p.JumpUser, p.JumpPassword, jumpAuthType, p.JumpAuthKeyPath, p.JumpAuthKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("jump host auth failed: %w", err)
	}
	jumpAddr := net.JoinHostPort(p.JumpHost, jumpPort)

	jumpClient, err := dialSSH(jumpAddr, jumpConfig)
	if err != nil {
		return nil, fmt.Errorf("jump host connection failed: %w", err)
	}

	targetConn, err := dialThroughJump(jumpClient, targetAddr)
	if err != nil {
		jumpClient.Close()
		return nil, fmt.Errorf("target connection through jump host failed: %w", err)
	}

	conn, chans, reqs, err := handshakeTarget(targetAddr, targetConfig, targetConn)
	if err != nil {
		targetConn.Close()
		jumpClient.Close()
		return nil, err
	}

	return &Client{
		conn:     ssh.NewClient(conn, chans, reqs),
		jumpConn: jumpClient,
	}, nil
}

func dialThroughJump(jumpClient *ssh.Client, targetAddr string) (net.Conn, error) {
	backoffs := []time.Duration{0, time.Second, 2 * time.Second}
	var lastErr error

	for attempt, wait := range backoffs {
		if wait > 0 {
			time.Sleep(wait)
			debug.Errorf("SSH jump tunnel retry %d/%d to %s: %v", attempt+1, maxDialAttempts, targetAddr, lastErr)
		}
		targetConn, err := jumpClient.Dial("tcp", targetAddr)
		if err != nil {
			lastErr = err
			if !isRetryableError(err) || attempt == len(backoffs)-1 {
				return nil, err
			}
			continue
		}
		return targetConn, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("jump tunnel dial failed for %s", targetAddr)
	}
	return nil, lastErr
}

func handshakeTarget(targetAddr string, targetConfig *ssh.ClientConfig, targetConn net.Conn) (ssh.Conn, <-chan ssh.NewChannel, <-chan *ssh.Request, error) {
	backoffs := []time.Duration{0, time.Second, 2 * time.Second}
	var lastErr error

	for attempt, wait := range backoffs {
		if wait > 0 {
			time.Sleep(wait)
			debug.Errorf("SSH target handshake retry %d/%d to %s: %v", attempt+1, maxDialAttempts, targetAddr, lastErr)
		}
		conn, chans, reqs, err := ssh.NewClientConn(targetConn, targetAddr, targetConfig)
		if err != nil {
			lastErr = err
			if !isRetryableError(err) || attempt == len(backoffs)-1 {
				return nil, nil, nil, fmt.Errorf("target ssh handshake through jump host failed: %w", err)
			}
			continue
		}
		return conn, chans, reqs, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("target ssh handshake failed for %s", targetAddr)
	}
	return nil, nil, nil, lastErr
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
