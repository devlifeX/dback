package ssh

import cryptossh "golang.org/x/crypto/ssh"

// Session is the common lifecycle interface for local and remote command sessions.
type Session interface {
	Close() error
	Wait() error
}

type remoteSession struct {
	s *cryptossh.Session
}

func wrapSession(s *cryptossh.Session) Session {
	return &remoteSession{s: s}
}

func (s *remoteSession) Close() error {
	return s.s.Close()
}

func (s *remoteSession) Wait() error {
	return s.s.Wait()
}
