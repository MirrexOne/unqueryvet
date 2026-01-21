// Package lsp provides LSP transport utilities.
package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// BaseReader reads LSP messages with Content-Length headers.
type BaseReader struct {
	reader *bufio.Reader
}

// NewBaseReader creates a new BaseReader.
func NewBaseReader(r io.Reader) *BaseReader {
	return &BaseReader{reader: bufio.NewReader(r)}
}

// Read reads a single LSP message, parsing Content-Length header.
func (r *BaseReader) Read() ([]byte, error) {
	// Read headers until empty line
	var contentLength int
	for {
		line, err := r.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			// End of headers
			break
		}

		// Parse Content-Length header
		if after, ok := strings.CutPrefix(line, "Content-Length:"); ok {
			value := strings.TrimSpace(after)
			contentLength, err = strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("invalid Content-Length: %v", err)
			}
		}
		// Ignore other headers (like Content-Type)
	}

	if contentLength == 0 {
		return nil, fmt.Errorf("missing Content-Length header")
	}

	// Read the message body
	body := make([]byte, contentLength)
	_, err := io.ReadFull(r.reader, body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// BaseWriter writes LSP messages with Content-Length headers.
type BaseWriter struct {
	writer io.Writer
}

// NewBaseWriter creates a new BaseWriter.
func NewBaseWriter(w io.Writer) *BaseWriter {
	return &BaseWriter{writer: w}
}

// Write writes a single LSP message with Content-Length header.
func (w *BaseWriter) Write(data []byte) error {
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	_, err := w.writer.Write([]byte(header))
	if err != nil {
		return err
	}
	_, err = w.writer.Write(data)
	return err
}

// WriteJSON marshals and writes a JSON message with Content-Length header.
func (w *BaseWriter) WriteJSON(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return w.Write(data)
}
