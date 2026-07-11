package store

import (
	"errors"

	"dback/models"

	"github.com/google/uuid"
)

var ErrNotifyChannelNotFound = errors.New("notify channel not found")

func (s *Store) ListNotifyChannels() ([]models.NotifyChannel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return nil, err
	}
	return append([]models.NotifyChannel(nil), s.notifyChannels...), nil
}

func (s *Store) GetNotifyChannel(id string) (models.NotifyChannel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return models.NotifyChannel{}, err
	}
	for _, ch := range s.notifyChannels {
		if ch.ID == id {
			return ch, nil
		}
	}
	return models.NotifyChannel{}, ErrNotifyChannelNotFound
}

func (s *Store) SaveNotifyChannel(ch models.NotifyChannel) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return err
	}
	if ch.ID == "" {
		ch.ID = uuid.NewString()
	}
	for i, existing := range s.notifyChannels {
		if existing.ID == ch.ID {
			s.notifyChannels[i] = ch
			s.bumpRevisionLocked()
			return s.persistVaultLocked()
		}
	}
	s.notifyChannels = append(s.notifyChannels, ch)
	s.bumpRevisionLocked()
	return s.persistVaultLocked()
}

func (s *Store) DeleteNotifyChannel(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return err
	}
	for i, ch := range s.notifyChannels {
		if ch.ID == id {
			s.notifyChannels = append(s.notifyChannels[:i], s.notifyChannels[i+1:]...)
			s.bumpRevisionLocked()
			return s.persistVaultLocked()
		}
	}
	return ErrNotifyChannelNotFound
}
