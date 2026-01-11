package audio

import (
	"fmt"
	"sync"

	"github.com/shashwatssp/deeprecall/internal/config"
	"github.com/shashwatssp/deeprecall/internal/models"
	"github.com/shashwatssp/deeprecall/internal/utils"
)

// Service manages audio input and output
type Service struct {
	cfg      *config.Config
	recorder *Recorder
	player   *Player
	mu       sync.RWMutex
	running  bool
}

// NewService creates a new audio service
func NewService(cfg *config.Config) (*Service, error) {
	recorder, err := NewRecorder(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create recorder: %w", err)
	}

	player, err := NewPlayer(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create player: %w", err)
	}

	return &Service{
		cfg:      cfg,
		recorder: recorder,
		player:   player,
	}, nil
}

// Start begins audio capture
func (s *Service) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("audio service already running")
	}

	if err := s.recorder.Start(); err != nil {
		return fmt.Errorf("failed to start recorder: %w", err)
	}

	s.running = true
	utils.GetLogger().Info("Audio service started")
	return nil
}

// Stop halts audio capture
func (s *Service) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	if err := s.recorder.Stop(); err != nil {
		return fmt.Errorf("failed to stop recorder: %w", err)
	}

	s.running = false
	utils.GetLogger().Info("Audio service stopped")
	return nil
}

// GetAudioStream returns a channel of audio chunks
func (s *Service) GetAudioStream() <-chan *models.AudioChunk {
	return s.recorder.GetStream()
}

// PlayAudio plays audio data through speakers
func (s *Service) PlayAudio(audioData []byte) error {
	return s.player.Play(audioData)
}
