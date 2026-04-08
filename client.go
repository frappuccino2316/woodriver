package woodriver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// transport handles raw HTTP communication with a WebDriver server.
type transport struct {
	base   string
	client *http.Client
}

type response struct {
	Value json.RawMessage `json:"value"`
}

type errorValue struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func (t *transport) do(method, path string, body any) (json.RawMessage, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, t.base+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var wr response
	if err := json.Unmarshal(raw, &wr); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// WebDriver errors are returned with 4xx/5xx status codes and an error key.
	if resp.StatusCode >= 400 {
		var ev errorValue
		if err := json.Unmarshal(wr.Value, &ev); err == nil && ev.Error != "" {
			return nil, &WebDriverError{Code: ev.Error, Message: ev.Message}
		}
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, raw)
	}

	return wr.Value, nil
}

func (t *transport) get(path string) (json.RawMessage, error) {
	return t.do(http.MethodGet, path, nil)
}

func (t *transport) post(path string, body any) (json.RawMessage, error) {
	return t.do(http.MethodPost, path, body)
}

func (t *transport) delete(path string) (json.RawMessage, error) {
	return t.do(http.MethodDelete, path, nil)
}
