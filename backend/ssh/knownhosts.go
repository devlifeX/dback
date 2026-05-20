package ssh

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

var (
	knownHostsPath string
	knownHostsMu   sync.Mutex
)

// SetKnownHostsFile configures the local TOFU known-hosts store path.
func SetKnownHostsFile(path string) {
	knownHostsMu.Lock()
	knownHostsPath = path
	knownHostsMu.Unlock()
}

func hostKeyCallback() (ssh.HostKeyCallback, error) {
	knownHostsMu.Lock()
	path := knownHostsPath
	knownHostsMu.Unlock()

	if path == "" {
		return nil, fmt.Errorf("SSH host key verification is required but no known_hosts file is configured")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("create known_hosts directory: %w", err)
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(path, nil, 0600); err != nil {
			return nil, fmt.Errorf("create known_hosts file: %w", err)
		}
	}

	verify, err := knownhosts.New(path)
	if err != nil {
		return nil, fmt.Errorf("load known_hosts: %w", err)
	}

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := verify(hostname, remote, key)
		if err == nil {
			return nil
		}
		var keyErr *knownhosts.KeyError
		if errors.As(err, &keyErr) && len(keyErr.Want) == 0 {
			return appendKnownHost(path, hostname, key)
		}
		return fmt.Errorf("host key verification failed for %s: %w", hostname, err)
	}, nil
}

func appendKnownHost(path, hostname string, key ssh.PublicKey) error {
	line := knownhosts.Line([]string{hostname}, key) + "\n"
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("append known host: %w", err)
	}
	defer f.Close()
	if _, err := f.WriteString(line); err != nil {
		return fmt.Errorf("write known host: %w", err)
	}
	return nil
}
