// Package transport handles raw HTTP communication with a WebDriver server.
// It is an internal package; external code must not import it directly.
package transport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Transport sends W3C WebDriver HTTP requests and decodes responses.
type Transport struct {
	Base   string
	Client *http.Client
}

// New returns a Transport pointed at base (e.g. "http://localhost:9515").
func New(base string, client *http.Client) *Transport {
	return &Transport{Base: base, Client: client}
}

type envelope struct {
	Value json.RawMessage `json:"value"`
}

type errorValue struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// Do executes an HTTP request and returns the unwrapped W3C value payload.
// 4xx/5xx responses are converted to a structured error.
func (t *Transport) Do(method, path string, body any) (json.RawMessage, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, t.Base+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := t.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var ev errorValue
		if err := json.Unmarshal(env.Value, &ev); err == nil && ev.Error != "" {
			// Return a sentinel value the woodriver package converts to *WebDriverError.
			return nil, &Error{Code: ev.Error, Message: ev.Message}
		}
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, raw)
	}

	return env.Value, nil
}

// Get is shorthand for Do(GET, path, nil).
func (t *Transport) Get(path string) (json.RawMessage, error) {
	return t.Do(http.MethodGet, path, nil)
}

// Post is shorthand for Do(POST, path, body).
func (t *Transport) Post(path string, body any) (json.RawMessage, error) {
	return t.Do(http.MethodPost, path, body)
}

// Delete is shorthand for Do(DELETE, path, nil).
func (t *Transport) Delete(path string) (json.RawMessage, error) {
	return t.Do(http.MethodDelete, path, nil)
}

// Error is a lightweight error type used by the transport layer to carry
// W3C error codes upward.  The woodriver package wraps these into its own
// *WebDriverError so callers only see the public type.
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("[%s] %s", e.Code, e.Message)
	}
	return fmt.Sprintf("[%s]", e.Code)
}
