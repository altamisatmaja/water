package capture

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Writer is a thread-safe append-only JSONL writer.
type Writer struct {
	mu   sync.Mutex
	file *os.File
	buf  *bufio.Writer
	path string
}

func NewWriter(eventsPath string) (*Writer, error) {
	f, err := os.OpenFile(eventsPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open events file: %w", err)
	}
	return &Writer{
		file: f,
		buf:  bufio.NewWriter(f),
		path: eventsPath,
	}, nil
}

func (w *Writer) Write(evt *Event) error {
	if evt.ID == "" {
		evt.ID = "evt-" + uuid.New().String()[:12]
	}
	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now().UTC()
	}

	data, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if _, err := w.buf.Write(data); err != nil {
		return fmt.Errorf("write event: %w", err)
	}
	if err := w.buf.WriteByte('\n'); err != nil {
		return fmt.Errorf("write newline: %w", err)
	}
	if err := w.buf.Flush(); err != nil {
		return fmt.Errorf("flush: %w", err)
	}
	return nil
}

func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	_ = w.buf.Flush()
	return w.file.Close()
}
