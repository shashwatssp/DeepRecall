package tts

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/shashwatssp/deeprecall/internal/config"
	"github.com/shashwatssp/deeprecall/internal/utils"
)

// GoogleTTS implements TTS using Google Translate TTS
type GoogleTTS struct {
	cfg      *config.Config
	cacheDir string
	client   *http.Client
}

// NewGoogleTTS creates a new Google TTS service
func NewGoogleTTS(cfg *config.Config) (*GoogleTTS, error) {
	cacheDir := cfg.TTS.CacheDir
	if cfg.TTS.CacheEnabled {
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create cache directory: %w", err)
		}
	}

	return &GoogleTTS{
		cfg:      cfg,
		cacheDir: cacheDir,
		client:   &http.Client{},
	}, nil
}

// Synthesize converts text to speech
func (g *GoogleTTS) Synthesize(text string) ([]byte, error) {
	logger := utils.GetLogger()

	if text == "" {
		return nil, fmt.Errorf("empty text")
	}

	// Check cache first
	if g.cfg.TTS.CacheEnabled {
		if cached, err := g.getFromCache(text); err == nil {
			logger.Debug("Using cached TTS audio")
			return cached, nil
		}
	}

	logger.Debugf("Synthesizing speech: %s", text)

	// Google Translate TTS URL
	baseURL := "https://translate.google.com/translate_tts"
	params := url.Values{}
	params.Add("ie", "UTF-8")
	params.Add("q", text)
	params.Add("tl", g.cfg.TTS.Language)
	params.Add("client", "tw-ob")

	fullURL := baseURL + "?" + params.Encode()

	// Make request
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("TTS request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TTS request failed with status: %d", resp.StatusCode)
	}

	// Read audio data
	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if g.cfg.TTS.CacheEnabled {
		if err := g.saveToCache(text, audioData); err != nil {
			logger.Warnf("Failed to cache TTS audio: %v", err)
		}
	}

	logger.Infof("Synthesized %d bytes of audio", len(audioData))
	return audioData, nil
}

// getFromCache retrieves cached audio
func (g *GoogleTTS) getFromCache(text string) ([]byte, error) {
	hash := g.textHash(text)
	cachePath := filepath.Join(g.cacheDir, hash+".mp3")

	return os.ReadFile(cachePath)
}

// saveToCache stores audio in cache
func (g *GoogleTTS) saveToCache(text string, audioData []byte) error {
	hash := g.textHash(text)
	cachePath := filepath.Join(g.cacheDir, hash+".mp3")

	return os.WriteFile(cachePath, audioData, 0644)
}

// textHash creates a hash of the text for caching
func (g *GoogleTTS) textHash(text string) string {
	hash := md5.Sum([]byte(text + g.cfg.TTS.Language))
	return hex.EncodeToString(hash[:])
}
