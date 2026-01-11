package context

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ledongthuc/pdf"
	"github.com/shashwatssp/deeprecall/internal/models"
	"github.com/shashwatssp/deeprecall/internal/utils"
)

type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

// ParseDocument parses a file and returns a Document
func (p *Parser) ParseDocument(filePath string) (*models.Document, error) {
	ext := strings.ToLower(filepath.Ext(filePath))

	var content string
	var err error

	switch ext {
	case ".pdf":
		content, err = p.parsePDF(filePath)
	case ".txt", ".md":
		content, err = p.parseText(filePath)
	default:
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}

	if err != nil {
		return nil, err
	}

	// Get file info
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	// Calculate hash
	hash, err := utils.FileHash(filePath)
	if err != nil {
		return nil, err
	}

	doc := &models.Document{
		ID:          hash,
		FilePath:    filePath,
		Content:     content,
		Hash:        hash,
		FileModTime: info.ModTime(),
		Metadata: map[string]string{
			"filename":  filepath.Base(filePath),
			"extension": ext,
			"size":      fmt.Sprintf("%d", info.Size()),
		},
	}

	return doc, nil
}

func (p *Parser) parsePDF(filePath string) (string, error) {
	file, reader, err := pdf.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open PDF: %w", err)
	}
	defer file.Close()

	var content strings.Builder
	numPages := reader.NumPage()

	for i := 1; i <= numPages; i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			utils.GetLogger().Warnf("Failed to extract text from page %d: %v", i, err)
			continue
		}

		content.WriteString(text)
		content.WriteString("\n")
	}

	return content.String(), nil
}

func (p *Parser) parseText(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var content strings.Builder
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		content.WriteString(scanner.Text())
		content.WriteString("\n")
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return content.String(), nil
}
