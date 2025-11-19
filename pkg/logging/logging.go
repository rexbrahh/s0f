package logging

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/rexliu/s0f/pkg/config"
)

// Logger wraps the standard log.Logger for now.
type Logger struct {
	*log.Logger
}

// New returns a logger writing to stdout until a richer logger is wired.
func New(prefix string) *Logger {
	return &Logger{Logger: log.New(os.Stdout, prefix+" ", log.LstdFlags|log.Lshortfile)}
}

// Configure applies logging settings from config.
func (l *Logger) Configure(cfg config.LoggingConfig) error {
	if l == nil || l.Logger == nil {
		return nil
	}
	if cfg.Level != "" {
		l.SetPrefix(strings.ToUpper(cfg.Level) + " " + l.Prefix())
	}
	if cfg.FilePath != "" {
		if err := os.MkdirAll(filepath.Dir(cfg.FilePath), 0o700); err != nil {
			return err
		}
		writer, err := newRollingFile(cfg.FilePath, cfg.FileMaxSize)
		if err != nil {
			return err
		}
		l.SetOutput(io.MultiWriter(os.Stdout, writer))
	}
	return nil
}

type rollingFile struct {
	path string
	max  int
	file *os.File
}

func newRollingFile(path string, maxMB int) (*rollingFile, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, err
	}
	return &rollingFile{path: path, max: maxMB, file: f}, nil
}

func (r *rollingFile) Write(p []byte) (int, error) {
	if r.max > 0 {
		if info, err := r.file.Stat(); err == nil && info.Size()+int64(len(p)) > int64(r.max)*1024*1024 {
			r.file.Close()
			os.Rename(r.path, r.path+".1")
			newFile, err := os.OpenFile(r.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
			if err != nil {
				return 0, err
			}
			r.file = newFile
		}
	}
	return r.file.Write(p)
}
