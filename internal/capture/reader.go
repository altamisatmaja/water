package capture

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

func ReadAll(path string) ([]*Event, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var events []*Event
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var evt Event
		if err := json.Unmarshal(line, &evt); err != nil {
			continue
		}
		events = append(events, &evt)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}
	return events, nil
}

// Tail follows a JSONL file, emitting newly appended events.
func Tail(ctx context.Context, path string, out chan<- *Event) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		return fmt.Errorf("seek end: %w", err)
	}

	reader := bufio.NewReader(f)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line, err := reader.ReadBytes('\n')
		if err == io.EOF {
			time.Sleep(200 * time.Millisecond)
			continue
		}
		if err != nil {
			return fmt.Errorf("read bytes: %w", err)
		}

		if len(line) == 0 {
			continue
		}
		line = line[:len(line)-1] // trim \n
		if len(line) == 0 {
			continue
		}

		var evt Event
		if err := json.Unmarshal(line, &evt); err != nil {
			continue
		}

		select {
		case out <- &evt:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
