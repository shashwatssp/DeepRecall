package stt

import (
	"fmt"

	"github.com/shashwatssp/deeprecall/internal/config"
	"github.com/shashwatssp/deeprecall/internal/models"
)

// Service provides speech-to-text functionality
type Service interface {
	Transcribe(audioData []byte) (*models.TranscriptionResult, error)
	TranscribeStream(stream <-chan []byte) (*models.TranscriptionResult, error)
}

// NewService creates a new STT service based on configuration
func NewService(cfg *config.Config) (Service, error) {
	switch cfg.STT.Provider {
	case "whisper":
		return NewWhisperService(cfg)
	case "google":
		return nil, fmt.Errorf("google STT not implemented yet")
	case "aws":
		return nil, fmt.Errorf("AWS transcribe not implemented yet")
	default:
		return nil, fmt.Errorf("unsupported STT provider: %s", cfg.STT.Provider)
	}
}
