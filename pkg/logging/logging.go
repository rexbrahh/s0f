package logging

import (
    "io"
    "log"
    "os"
)

// Logger wraps the standard log.Logger for now.
type Logger struct {
    *log.Logger
}

// New returns a logger writing to stdout until a richer logger is wired.
func New(prefix string) *Logger {
    return &Logger{Logger: log.New(os.Stdout, prefix+" ", log.LstdFlags|log.Lshortfile)}
}

// WithOutput switches the logger output.
func (l *Logger) WithOutput(w io.Writer) {
    if l == nil || l.Logger == nil {
        return
    }
    l.SetOutput(w)
}
