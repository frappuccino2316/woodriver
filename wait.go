package woodriver

import (
	"errors"
	"strings"
	"time"
)

// Condition is a function that checks some state of a session.
// It returns (true, nil) when the condition is met, (false, nil) to retry,
// or (false, err) to abort immediately.
type Condition func(Session) (bool, error)

// Waiter provides explicit wait support.
type Waiter struct {
	sess     Session
	timeout  time.Duration
	interval time.Duration
}

// Until polls cond until it returns true or the timeout elapses.
func (w *Waiter) Until(cond Condition) error {
	deadline := time.Now().Add(w.timeout)
	for {
		ok, err := cond(w.sess)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		if time.Now().After(deadline) {
			return ErrTimeout
		}
		time.Sleep(w.interval)
	}
}

// UntilElement waits until an element is present in the DOM and returns it.
func (w *Waiter) UntilElement(by By, value string) (Element, error) {
	var found Element
	err := w.Until(func(s Session) (bool, error) {
		el, err := s.FindElement(by, value)
		if err != nil {
			if errors.Is(err, ErrNoSuchElement) {
				return false, nil
			}
			return false, err
		}
		found = el
		return true, nil
	})
	return found, err
}

// ElementVisible waits until the element is present and displayed.
func ElementVisible(by By, value string) Condition {
	return func(s Session) (bool, error) {
		el, err := s.FindElement(by, value)
		if err != nil {
			if errors.Is(err, ErrNoSuchElement) {
				return false, nil
			}
			return false, err
		}
		return el.IsDisplayed()
	}
}

// ElementClickable waits until the element is displayed and enabled.
func ElementClickable(by By, value string) Condition {
	return func(s Session) (bool, error) {
		el, err := s.FindElement(by, value)
		if err != nil {
			if errors.Is(err, ErrNoSuchElement) {
				return false, nil
			}
			return false, err
		}
		displayed, err := el.IsDisplayed()
		if err != nil || !displayed {
			return false, err
		}
		return el.IsEnabled()
	}
}

// TitleContains waits until the page title contains substr.
func TitleContains(substr string) Condition {
	return func(s Session) (bool, error) {
		title, err := s.Title()
		if err != nil {
			return false, err
		}
		return strings.Contains(title, substr), nil
	}
}

// URLMatches waits until the current URL contains substr.
func URLMatches(substr string) Condition {
	return func(s Session) (bool, error) {
		url, err := s.CurrentURL()
		if err != nil {
			return false, err
		}
		return strings.Contains(url, substr), nil
	}
}
