package woodriver

import (
	"encoding/json"
	"time"
)

// Cookie represents an HTTP cookie in a browser session.
type Cookie struct {
	Name     string    `json:"name"`
	Value    string    `json:"value"`
	Domain   string    `json:"domain,omitempty"`
	Path     string    `json:"path,omitempty"`
	Secure   bool      `json:"secure,omitempty"`
	HTTPOnly bool      `json:"httpOnly,omitempty"`
	SameSite string    `json:"sameSite,omitempty"` // "Strict", "Lax", "None"
	Expiry   time.Time `json:"-"`                  // serialized separately
}

// cookieJSON is the on-wire representation used by W3C WebDriver.
type cookieJSON struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Domain   string `json:"domain,omitempty"`
	Path     string `json:"path,omitempty"`
	Secure   bool   `json:"secure,omitempty"`
	HTTPOnly bool   `json:"httpOnly,omitempty"`
	SameSite string `json:"sameSite,omitempty"`
	Expiry   *int64 `json:"expiry,omitempty"` // Unix seconds
}

func (c Cookie) toJSON() cookieJSON {
	cj := cookieJSON{
		Name:     c.Name,
		Value:    c.Value,
		Domain:   c.Domain,
		Path:     c.Path,
		Secure:   c.Secure,
		HTTPOnly: c.HTTPOnly,
		SameSite: c.SameSite,
	}
	if !c.Expiry.IsZero() {
		exp := c.Expiry.Unix()
		cj.Expiry = &exp
	}
	return cj
}

func cookieFromJSON(cj cookieJSON) Cookie {
	c := Cookie{
		Name:     cj.Name,
		Value:    cj.Value,
		Domain:   cj.Domain,
		Path:     cj.Path,
		Secure:   cj.Secure,
		HTTPOnly: cj.HTTPOnly,
		SameSite: cj.SameSite,
	}
	if cj.Expiry != nil {
		c.Expiry = time.Unix(*cj.Expiry, 0)
	}
	return c
}

// Cookies returns all cookies visible to the current page.
func (s *session) Cookies() ([]Cookie, error) {
	raw, err := s.get(s.path("/cookie"))
	if err != nil {
		return nil, err
	}
	var cjs []cookieJSON
	if err := json.Unmarshal(raw, &cjs); err != nil {
		return nil, err
	}
	cookies := make([]Cookie, len(cjs))
	for i, cj := range cjs {
		cookies[i] = cookieFromJSON(cj)
	}
	return cookies, nil
}

// Cookie returns the named cookie.
func (s *session) Cookie(name string) (Cookie, error) {
	raw, err := s.get(s.path("/cookie/" + name))
	if err != nil {
		return Cookie{}, err
	}
	var cj cookieJSON
	if err := json.Unmarshal(raw, &cj); err != nil {
		return Cookie{}, err
	}
	return cookieFromJSON(cj), nil
}

// AddCookie adds a cookie to the current page's domain.
func (s *session) AddCookie(c Cookie) error {
	_, err := s.post(s.path("/cookie"), map[string]any{"cookie": c.toJSON()})
	return err
}

// DeleteCookie removes the named cookie.
func (s *session) DeleteCookie(name string) error {
	_, err := s.delete(s.path("/cookie/" + name))
	return err
}

// DeleteAllCookies removes all cookies visible to the current page.
func (s *session) DeleteAllCookies() error {
	_, err := s.delete(s.path("/cookie"))
	return err
}
