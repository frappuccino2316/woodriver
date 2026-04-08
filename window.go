package woodriver

import (
	"encoding/json"
	"fmt"
)

// WindowType is used when opening a new browsing context.
type WindowType string

const (
	WindowTypeTab    WindowType = "tab"
	WindowTypeWindow WindowType = "window"
)

// WindowHandle wraps a browser window or tab handle.
type WindowHandle struct {
	Handle string `json:"handle"`
	Type   string `json:"type"`
}

// --- Window management (added to Session via windowOps) ---
// These methods are defined on *session and exposed through the extended
// interface below so callers can use them without type assertions.

// WindowOps extends Session with window/frame management.
type WindowOps interface {
	Session

	// Window handles
	CurrentWindowHandle() (string, error)
	WindowHandles() ([]string, error)
	SwitchToWindow(handle string) error
	NewWindow(t WindowType) (WindowHandle, error)

	// Window state
	Maximize() error
	Minimize() error
	Fullscreen() error

	// Frame navigation
	SwitchToFrame(id any) error
	SwitchToParentFrame() error

	// Alert / dialog
	AcceptAlert() error
	DismissAlert() error
	AlertText() (string, error)
	SendAlertText(text string) error

	// Cookies
	Cookies() ([]Cookie, error)
	Cookie(name string) (Cookie, error)
	AddCookie(c Cookie) error
	DeleteCookie(name string) error
	DeleteAllCookies() error

	// Actions
	Actions() *Actions
}

// Ensure *session satisfies WindowOps.
var _ WindowOps = (*session)(nil)

// --- window handle methods ---

func (s *session) CurrentWindowHandle() (string, error) {
	raw, err := s.get(s.path("/window"))
	if err != nil {
		return "", err
	}
	var h string
	return h, json.Unmarshal(raw, &h)
}

func (s *session) WindowHandles() ([]string, error) {
	raw, err := s.get(s.path("/window/handles"))
	if err != nil {
		return nil, err
	}
	var handles []string
	return handles, json.Unmarshal(raw, &handles)
}

func (s *session) SwitchToWindow(handle string) error {
	_, err := s.post(s.path("/window"), map[string]any{"handle": handle})
	return err
}

func (s *session) NewWindow(t WindowType) (WindowHandle, error) {
	raw, err := s.post(s.path("/window/new"), map[string]any{"type": string(t)})
	if err != nil {
		return WindowHandle{}, err
	}
	var wh WindowHandle
	return wh, json.Unmarshal(raw, &wh)
}

// --- window state ---

func (s *session) Maximize() error {
	_, err := s.post(s.path("/window/maximize"), map[string]any{})
	return err
}

func (s *session) Minimize() error {
	_, err := s.post(s.path("/window/minimize"), map[string]any{})
	return err
}

func (s *session) Fullscreen() error {
	_, err := s.post(s.path("/window/fullscreen"), map[string]any{})
	return err
}

// --- frame navigation ---

// SwitchToFrame switches focus to a frame.
// id may be:
//   - nil          → top-level browsing context
//   - int          → frame index
//   - Element      → frame element
func (s *session) SwitchToFrame(id any) error {
	var payload map[string]any
	switch v := id.(type) {
	case nil:
		payload = map[string]any{"id": nil}
	case int:
		payload = map[string]any{"id": v}
	case Element:
		e, ok := v.(*element)
		if !ok {
			return fmt.Errorf("SwitchToFrame: unsupported Element implementation")
		}
		payload = map[string]any{
			"id": map[string]string{webElementKey: e.id},
		}
	default:
		return fmt.Errorf("SwitchToFrame: unsupported id type %T", id)
	}
	_, err := s.post(s.path("/frame"), payload)
	return err
}

func (s *session) SwitchToParentFrame() error {
	_, err := s.post(s.path("/frame/parent"), map[string]any{})
	return err
}

// --- alert / dialog ---

func (s *session) AcceptAlert() error {
	_, err := s.post(s.path("/alert/accept"), map[string]any{})
	return err
}

func (s *session) DismissAlert() error {
	_, err := s.post(s.path("/alert/dismiss"), map[string]any{})
	return err
}

func (s *session) AlertText() (string, error) {
	raw, err := s.get(s.path("/alert/text"))
	if err != nil {
		return "", err
	}
	var text string
	return text, json.Unmarshal(raw, &text)
}

func (s *session) SendAlertText(text string) error {
	_, err := s.post(s.path("/alert/text"), map[string]any{"text": text})
	return err
}
