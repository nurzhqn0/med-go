package logger

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"time"
)

type EventLogger struct {
	writer io.Writer
}

func New() *EventLogger {
	return &EventLogger{writer: os.Stdout}
}

func NewWithWriter(writer io.Writer) *EventLogger {
	return &EventLogger{writer: writer}
}

func (l *EventLogger) Log(subject string, event map[string]any) {
	line := struct {
		Time    string         `json:"time"`
		Subject string         `json:"subject"`
		Event   map[string]any `json:"event"`
	}{
		Time:    time.Now().UTC().Format(time.RFC3339),
		Subject: subject,
		Event:   event,
	}

	if err := json.NewEncoder(l.writer).Encode(line); err != nil {
		log.Printf("failed to write notification log subject=%s: %v", subject, err)
	}
}
