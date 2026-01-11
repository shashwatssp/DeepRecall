package context

import (
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/shashwatssp/deeprecall/internal/config"
	"github.com/shashwatssp/deeprecall/internal/utils"
)

type Watcher struct {
	indexer       *Indexer
	cfg           *config.Config
	watcher       *fsnotify.Watcher
	onIndexUpdate func(filePath string)
}

func NewWatcher(indexer *Indexer, cfg *config.Config) (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		indexer: indexer,
		cfg:     cfg,
		watcher: watcher,
	}, nil
}

// Start begins watching the context directory
func (w *Watcher) Start() error {
	logger := utils.GetLogger()

	// Add context directory to watcher
	if err := w.watcher.Add(w.cfg.Context.Folder); err != nil {
		return err
	}

	logger.Infof("Watching context folder: %s", w.cfg.Context.Folder)

	go w.watch()
	return nil
}

// Stop stops the watcher
func (w *Watcher) Stop() error {
	return w.watcher.Close()
}

// OnIndexUpdate sets callback for when files are reindexed
func (w *Watcher) OnIndexUpdate(callback func(string)) {
	w.onIndexUpdate = callback
}

func (w *Watcher) watch() {
	logger := utils.GetLogger()
	debounce := make(map[string]*time.Timer)

	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Only handle write and create events
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				ext := filepath.Ext(event.Name)
				supported := false
				for _, supportedExt := range w.cfg.Context.SupportedExtensions {
					if ext == supportedExt {
						supported = true
						break
					}
				}

				if !supported {
					continue
				}

				logger.Infof("Detected change in file: %s", event.Name)

				// Debounce: wait for file to finish writing
				if timer, exists := debounce[event.Name]; exists {
					timer.Stop()
				}

				debounce[event.Name] = time.AfterFunc(2*time.Second, func() {
					w.handleFileChange(event.Name)
					delete(debounce, event.Name)
				})
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			logger.Errorf("Watcher error: %v", err)
		}
	}
}

func (w *Watcher) handleFileChange(filePath string) {
	logger := utils.GetLogger()
	logger.Infof("Reindexing file: %s", filePath)

	_, err := w.indexer.IndexFile(filePath, true)
	if err != nil {
		logger.Errorf("Failed to reindex %s: %v", filePath, err)
		return
	}

	logger.Infof("Successfully reindexed: %s", filePath)

	if w.onIndexUpdate != nil {
		w.onIndexUpdate(filePath)
	}
}
