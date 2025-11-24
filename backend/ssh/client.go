package ssh

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"time"

	"dback/models"

	"golang.org/x/crypto/ssh"
)

// Client wraps the ssh.Client and provides high-level operations
type Client struct {
	conn *ssh.Client
}

// NewClient creates a new SSH client based on the profile
func NewClient(p models.Profile) (*Client, error) {
	config := &ssh.ClientConfig{
		User: p.SSHUser,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // For simplicity; in prod, verify host keys
		Timeout:         10 * time.Second,
	}

	if p.AuthType == models.AuthTypePassword {
		// Need to prompt or have password in profile? 
		// The requirements imply saving credentials, so we assume it's in the profile or user prompts?
		// The Profile struct in models.go has DBPassword but not SSHPassword.
		// Wait, requirement says: "Auth method path, DB creds". 
		// Usually SSH password isn't saved for security or is keyed in.
		// For this MVP, let's assume we might need to add SSHPassword to Profile or prompt.
		// Revisiting requirements: "Auth Type selector (Password vs. Key File entry)."
		// If "Password" is selected, there should be a password field. 
		// I'll assume for now we might have an SSHPassword field or similar.
		// Let's check models.go... It's missing SSHPassword. I should probably add it or assume KeyFile is preferred.
		// But "Password authentication" is a requirement.
		// I will update models.go later or handling it by adding SSHPassword to the struct if I missed it.
		// Let's look at the previous models.go content... Yes, SSHPassword is missing.
		// I will proceed assuming I can add it, or I'll stick to KeyFile for now and fix later.
		// Actually, I'll use a placeholder or if I can't edit models now, I'll handle it.
		// Just checked models.go again, indeed missing. I will assume for this step that I'll update models.go next.
		config.Auth = []ssh.AuthMethod{
			ssh.Password(p.SSHPassword),
		}
	} else {
		// Key File
		key, err := ioutil.ReadFile(p.AuthKeyPath)
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

	addr := net.JoinHostPort(p.Host, p.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, err
	}

	return &Client{conn: client}, nil
}

// Close closes the SSH connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// RunCommandStream executes a command and returns its stdout pipe.
// This is crucial for streaming large dumps.
func (c *Client) RunCommandStream(cmd string) (io.Reader, *ssh.Session, error) {
	session, err := c.conn.NewSession()
	if err != nil {
		return nil, nil, err
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		return nil, nil, err
	}

	// We also need to capture stderr to report errors
	// For simplicity, we might log it or pipe it elsewhere
	// stderr, _ := session.StderrPipe() 

	if err := session.Start(cmd); err != nil {
		session.Close()
		return nil, nil, err
	}

	return stdout, session, nil
}

// RunCommandPipeInput executes a command and returns its stdin pipe.
// This is used for uploading/restoring dumps.
func (c *Client) RunCommandPipeInput(cmd string) (io.WriteCloser, *ssh.Session, error) {
	session, err := c.conn.NewSession()
	if err != nil {
		return nil, nil, err
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		session.Close()
		return nil, nil, err
	}

	if err := session.Start(cmd); err != nil {
		session.Close()
		return nil, nil, err
	}

	return stdin, session, nil
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
