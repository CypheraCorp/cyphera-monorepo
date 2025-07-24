package middleware

import (
	"bytes"
	"io"
)

// BodyReader implements io.ReadCloser to allow re-reading request body
type BodyReader struct {
	*bytes.Reader
}

// NewBodyReader creates a new BodyReader from bytes
func NewBodyReader(body []byte) io.ReadCloser {
	return &BodyReader{Reader: bytes.NewReader(body)}
}

// Close implements io.ReadCloser
func (r *BodyReader) Close() error {
	return nil
}
