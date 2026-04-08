package woodriver

import (
	"encoding/json"
	"fmt"
)

// webElementKey is the W3C WebDriver element identifier key.
const webElementKey = "element-6066-11e4-a52e-4f735466cecf"

// element implements Element backed by a WebDriver session.
type element struct {
	id   string
	sess *session
}

// Element represents a DOM element within a browser session.
type Element interface {
	Click() error
	SendKeys(text string) error
	Clear() error
	Text() (string, error)
	Attribute(name string) (string, error)
	Property(name string) (any, error)
	IsDisplayed() (bool, error)
	IsEnabled() (bool, error)
	IsSelected() (bool, error)
	TagName() (string, error)
	Rect() (Rect, error)
	FindElement(by By, value string) (Element, error)
	FindElements(by By, value string) ([]Element, error)
}

func (e *element) path(suffix string) string {
	return fmt.Sprintf("/session/%s/element/%s%s", e.sess.id, e.id, suffix)
}

func (e *element) Click() error {
	_, err := e.sess.t.post(e.path("/click"), map[string]any{})
	return err
}

func (e *element) SendKeys(text string) error {
	_, err := e.sess.t.post(e.path("/value"), map[string]any{"text": text})
	return err
}

func (e *element) Clear() error {
	_, err := e.sess.t.post(e.path("/clear"), map[string]any{})
	return err
}

func (e *element) Text() (string, error) {
	raw, err := e.sess.t.get(e.path("/text"))
	if err != nil {
		return "", err
	}
	var s string
	return s, json.Unmarshal(raw, &s)
}

func (e *element) Attribute(name string) (string, error) {
	raw, err := e.sess.t.get(e.path("/attribute/" + name))
	if err != nil {
		return "", err
	}
	var s string
	return s, json.Unmarshal(raw, &s)
}

func (e *element) Property(name string) (any, error) {
	raw, err := e.sess.t.get(e.path("/property/" + name))
	if err != nil {
		return nil, err
	}
	var v any
	return v, json.Unmarshal(raw, &v)
}

func (e *element) IsDisplayed() (bool, error) {
	raw, err := e.sess.t.get(e.path("/displayed"))
	if err != nil {
		return false, err
	}
	var b bool
	return b, json.Unmarshal(raw, &b)
}

func (e *element) IsEnabled() (bool, error) {
	raw, err := e.sess.t.get(e.path("/enabled"))
	if err != nil {
		return false, err
	}
	var b bool
	return b, json.Unmarshal(raw, &b)
}

func (e *element) IsSelected() (bool, error) {
	raw, err := e.sess.t.get(e.path("/selected"))
	if err != nil {
		return false, err
	}
	var b bool
	return b, json.Unmarshal(raw, &b)
}

func (e *element) TagName() (string, error) {
	raw, err := e.sess.t.get(e.path("/name"))
	if err != nil {
		return "", err
	}
	var s string
	return s, json.Unmarshal(raw, &s)
}

func (e *element) Rect() (Rect, error) {
	raw, err := e.sess.t.get(e.path("/rect"))
	if err != nil {
		return Rect{}, err
	}
	var r Rect
	return r, json.Unmarshal(raw, &r)
}

func (e *element) FindElement(by By, value string) (Element, error) {
	return findElement(e.sess, e.path("/element"), by, value)
}

func (e *element) FindElements(by By, value string) ([]Element, error) {
	return findElements(e.sess, e.path("/elements"), by, value)
}

// findElement performs a POST to locate a single element.
func findElement(sess *session, path string, by By, value string) (Element, error) {
	raw, err := sess.t.post(path, map[string]any{"using": string(by), "value": value})
	if err != nil {
		return nil, err
	}
	id, err := extractElementID(raw)
	if err != nil {
		return nil, err
	}
	return &element{id: id, sess: sess}, nil
}

// findElements performs a POST to locate multiple elements.
func findElements(sess *session, path string, by By, value string) ([]Element, error) {
	raw, err := sess.t.post(path, map[string]any{"using": string(by), "value": value})
	if err != nil {
		return nil, err
	}
	var items []map[string]string
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("decode elements: %w", err)
	}
	els := make([]Element, 0, len(items))
	for _, item := range items {
		id, ok := item[webElementKey]
		if !ok {
			return nil, fmt.Errorf("missing element key in response")
		}
		els = append(els, &element{id: id, sess: sess})
	}
	return els, nil
}

func extractElementID(raw json.RawMessage) (string, error) {
	var m map[string]string
	if err := json.Unmarshal(raw, &m); err != nil {
		return "", fmt.Errorf("decode element: %w", err)
	}
	id, ok := m[webElementKey]
	if !ok {
		return "", fmt.Errorf("missing element key in response")
	}
	return id, nil
}
