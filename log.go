package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type contextKey int

// ContextKeyLog is used to access an http.Request's *Log from its context
const ContextKeyLog contextKey = iota

// Error is a wrapper for error to marshal properly
type Error struct {
	error
}

// MarshalJSON implements json.Marshaler
func (e *Error) MarshalJSON() ([]byte, error) {
	if e.error == nil {
		return nil, nil
	}
	return json.Marshal(e.Error())
}

// Log is a log entry
type Log struct {
	Level            string    `json:"level"`
	Time             time.Time `json:"time"`
	IP               string    `json:"ip"`
	Method           string    `json:"method"`
	URL              string    `json:"url"`
	Status           int       `json:"status"`
	Size             int       `json:"size"`
	ClientIdentifier string    `json:"client_identifier,omitempty"`
	Error            *Error    `json:"error,omitempty"`
}

type statusWriter struct {
	http.ResponseWriter
	*Log
}

func (s *statusWriter) Write(b []byte) (int, error) {
	n, err := s.ResponseWriter.Write(b)
	s.Log.Size += n
	return n, err
}

func (s *statusWriter) WriteHeader(statusCode int) {
	s.Log.Status = statusCode
	s.ResponseWriter.WriteHeader(statusCode)
}

// Logger logs Log entries and make sures lines aren't interspersed
type Logger struct {
	io.WriteCloser
	mu *sync.Mutex
}

// NewLogger returns a new Logger. You should defer the Close method on the returns logger to make sure all writes are flushed
func NewLogger(w io.WriteCloser) *Logger {
	return &Logger{WriteCloser: w, mu: new(sync.Mutex)}
}

// Write writes the given log entry
func (l *Logger) Write(lg *Log) {
	l.mu.Lock()
	defer l.mu.Unlock()

	enc := json.NewEncoder(l.WriteCloser)
	if err := enc.Encode(lg); err != nil {
		log.Println("could not encode log:", err)
	}
}

// LogHandler is http middleware that logs requests
func LogHandler(logger *Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := &Log{
			Level:  "info",
			Time:   time.Now(),
			IP:     strings.Split(r.RemoteAddr, ":")[0],
			Method: r.Method,
			URL:    r.URL.String(),
			Status: 200,
		}

		r = r.WithContext(context.WithValue(r.Context(), ContextKeyLog, l))
		next.ServeHTTP(&statusWriter{ResponseWriter: w, Log: l}, r)

		if l.Status >= 500 {
			l.Level = "error"
		} else if l.Status >= 400 {
			l.Level = "warn"
		}

		logger.Write(l)
	})
}
