package woodriver

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/frappuccino2316/woodriver/internal/transport"
)

const defaultPollInterval = 200 * time.Millisecond

// Session represents a browser session.
type Session interface {
	// Navigation
	Navigate(url string) error
	CurrentURL() (string, error)
	Title() (string, error)
	Back() error
	Forward() error
	Refresh() error

	// Element search
	FindElement(by By, value string) (Element, error)
	FindElements(by By, value string) ([]Element, error)

	// JavaScript
	Execute(script string, args ...any) (any, error)
	ExecuteAsync(script string, args ...any) (any, error)

	// Window
	Screenshot() ([]byte, error)
	WindowRect() (Rect, error)
	SetWindowRect(rect Rect) error

	// Explicit wait
	Wait(timeout time.Duration) *Waiter

	// Lifecycle
	Close() error
	Quit() error
}

// session is the concrete implementation of Session.
type session struct {
	id string
	t  *transport.Transport
}

func (s *session) path(suffix string) string {
	return fmt.Sprintf("/session/%s%s", s.id, suffix)
}

// liftErr converts a transport.Error into a *WebDriverError so callers only
// deal with the public error type.
func liftErr(err error) error {
	var te *transport.Error
	if errors.As(err, &te) {
		return &WebDriverError{Code: te.Code, Message: te.Message}
	}
	return err
}

func (s *session) get(path string) (json.RawMessage, error) {
	raw, err := s.t.Get(path)
	return raw, liftErr(err)
}

func (s *session) post(path string, body any) (json.RawMessage, error) {
	raw, err := s.t.Post(path, body)
	return raw, liftErr(err)
}

func (s *session) delete(path string) (json.RawMessage, error) {
	raw, err := s.t.Delete(path)
	return raw, liftErr(err)
}

// Navigate navigates to the given URL.
func (s *session) Navigate(url string) error {
	_, err := s.post(s.path("/url"), map[string]any{"url": url})
	return err
}

// CurrentURL returns the current page URL.
func (s *session) CurrentURL() (string, error) {
	raw, err := s.get(s.path("/url"))
	if err != nil {
		return "", err
	}
	var v string
	return v, json.Unmarshal(raw, &v)
}

// Title returns the current page title.
func (s *session) Title() (string, error) {
	raw, err := s.get(s.path("/title"))
	if err != nil {
		return "", err
	}
	var v string
	return v, json.Unmarshal(raw, &v)
}

// Back navigates backward in the browser history.
func (s *session) Back() error {
	_, err := s.post(s.path("/back"), map[string]any{})
	return err
}

// Forward navigates forward in the browser history.
func (s *session) Forward() error {
	_, err := s.post(s.path("/forward"), map[string]any{})
	return err
}

// Refresh reloads the current page.
func (s *session) Refresh() error {
	_, err := s.post(s.path("/refresh"), map[string]any{})
	return err
}

// FindElement locates a single element using the given strategy.
func (s *session) FindElement(by By, value string) (Element, error) {
	return findElement(s, s.path("/element"), by, value)
}

// FindElements locates all matching elements.
func (s *session) FindElements(by By, value string) ([]Element, error) {
	return findElements(s, s.path("/elements"), by, value)
}

// Execute runs a synchronous JavaScript snippet.
func (s *session) Execute(script string, args ...any) (any, error) {
	if args == nil {
		args = []any{}
	}
	raw, err := s.post(s.path("/execute/sync"), map[string]any{
		"script": script,
		"args":   args,
	})
	if err != nil {
		return nil, err
	}
	var v any
	return v, json.Unmarshal(raw, &v)
}

// ExecuteAsync runs an asynchronous JavaScript snippet.
func (s *session) ExecuteAsync(script string, args ...any) (any, error) {
	if args == nil {
		args = []any{}
	}
	raw, err := s.post(s.path("/execute/async"), map[string]any{
		"script": script,
		"args":   args,
	})
	if err != nil {
		return nil, err
	}
	var v any
	return v, json.Unmarshal(raw, &v)
}

// Screenshot captures the current viewport as a PNG and returns the raw bytes.
func (s *session) Screenshot() ([]byte, error) {
	raw, err := s.get(s.path("/screenshot"))
	if err != nil {
		return nil, err
	}
	var encoded string
	if err := json.Unmarshal(raw, &encoded); err != nil {
		return nil, fmt.Errorf("decode screenshot: %w", err)
	}
	return base64.StdEncoding.DecodeString(encoded)
}

// WindowRect returns the current window position and size.
func (s *session) WindowRect() (Rect, error) {
	raw, err := s.get(s.path("/window/rect"))
	if err != nil {
		return Rect{}, err
	}
	var r Rect
	return r, json.Unmarshal(raw, &r)
}

// SetWindowRect sets the window position and size.
func (s *session) SetWindowRect(rect Rect) error {
	_, err := s.post(s.path("/window/rect"), rect)
	return err
}

// Wait returns a Waiter configured with the given timeout.
func (s *session) Wait(timeout time.Duration) *Waiter {
	return &Waiter{sess: s, timeout: timeout, interval: defaultPollInterval}
}

// Close closes the current browser window (not the session).
func (s *session) Close() error {
	_, err := s.delete(s.path("/window"))
	return err
}

// Quit deletes the entire session.
func (s *session) Quit() error {
	_, err := s.delete(s.path(""))
	return err
}

// Driver creates and manages browser sessions.
type Driver struct {
	t *transport.Transport
}

// Option configures a Driver.
type Option func(*Driver)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(c *http.Client) Option {
	return func(d *Driver) { d.t.Client = c }
}

// New creates a Driver that communicates with the WebDriver server at driverURL.
func New(driverURL string, opts ...Option) *Driver {
	d := &Driver{
		t: transport.New(driverURL, &http.Client{Timeout: 30 * time.Second}),
	}
	for _, o := range opts {
		o(d)
	}
	return d
}

// NewSession starts a new browser session with the given capabilities.
// The returned interface satisfies both Session and WindowOps.
func (d *Driver) NewSession(caps Capabilities) (WindowOps, error) {
	raw, err := d.t.Post("/session", caps.toW3C())
	if err != nil {
		return nil, liftErr(err)
	}

	var result struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("decode session: %w", err)
	}
	if result.SessionID == "" {
		return nil, fmt.Errorf("empty session id in response")
	}
	return &session{id: result.SessionID, t: d.t}, nil
}
