package woodriver

// Actions builds a W3C Actions payload through a fluent API.
//
// Usage:
//
//	err := sess.Actions().
//	    MouseMove(100, 200).
//	    MouseClick(MouseLeft).
//	    KeyDown(woodriver.KeyControl).
//	    KeySendKeys("a").
//	    KeyUp(woodriver.KeyControl).
//	    Perform()

// MouseButton identifies a pointer button.
type MouseButton int

const (
	MouseLeft   MouseButton = 0
	MouseMiddle MouseButton = 1
	MouseRight  MouseButton = 2
)

// Actions is a fluent builder for W3C input action sequences.
type Actions struct {
	sess    *session
	pointer []map[string]any
	keys    []map[string]any
	wheel   []map[string]any
}

// Actions returns an Actions builder for this session.
func (s *session) Actions() *Actions {
	return &Actions{sess: s}
}

// --- Pointer (mouse) actions ---

// MouseMove moves the pointer to an absolute viewport coordinate.
func (a *Actions) MouseMove(x, y float64) *Actions {
	a.pointer = append(a.pointer, map[string]any{
		"type":     "pointerMove",
		"duration": 0,
		"x":        x,
		"y":        y,
		"origin":   "viewport",
	})
	return a
}

// MouseMoveToElement moves the pointer to the centre of an element.
func (a *Actions) MouseMoveToElement(el Element) *Actions {
	e, ok := el.(*element)
	if !ok {
		return a
	}
	a.pointer = append(a.pointer, map[string]any{
		"type":     "pointerMove",
		"duration": 0,
		"x":        0,
		"y":        0,
		"origin":   map[string]string{webElementKey: e.id},
	})
	return a
}

// MouseDown presses a mouse button.
func (a *Actions) MouseDown(btn MouseButton) *Actions {
	a.pointer = append(a.pointer, map[string]any{
		"type":   "pointerDown",
		"button": int(btn),
	})
	return a
}

// MouseUp releases a mouse button.
func (a *Actions) MouseUp(btn MouseButton) *Actions {
	a.pointer = append(a.pointer, map[string]any{
		"type":   "pointerUp",
		"button": int(btn),
	})
	return a
}

// MouseClick presses and releases a button (single click).
func (a *Actions) MouseClick(btn MouseButton) *Actions {
	return a.MouseDown(btn).MouseUp(btn)
}

// MouseDoubleClick sends two click sequences.
func (a *Actions) MouseDoubleClick(btn MouseButton) *Actions {
	return a.MouseClick(btn).MouseClick(btn)
}

// ClickElement moves to an element and left-clicks it.
func (a *Actions) ClickElement(el Element) *Actions {
	return a.MouseMoveToElement(el).MouseClick(MouseLeft)
}

// --- Keyboard actions ---

// KeyDown presses a key (use Key* constants for special keys).
func (a *Actions) KeyDown(key string) *Actions {
	a.keys = append(a.keys, map[string]any{
		"type":  "keyDown",
		"value": key,
	})
	return a
}

// KeyUp releases a key.
func (a *Actions) KeyUp(key string) *Actions {
	a.keys = append(a.keys, map[string]any{
		"type":  "keyUp",
		"value": key,
	})
	return a
}

// KeySendKeys types a string by pressing and releasing each rune in sequence.
func (a *Actions) KeySendKeys(text string) *Actions {
	for _, ch := range text {
		s := string(ch)
		a.keys = append(a.keys,
			map[string]any{"type": "keyDown", "value": s},
			map[string]any{"type": "keyUp", "value": s},
		)
	}
	return a
}

// --- Wheel (scroll) actions ---

// Scroll dispatches a wheel scroll at viewport coordinates (x, y).
// deltaX and deltaY are in CSS pixels.
func (a *Actions) Scroll(x, y, deltaX, deltaY int) *Actions {
	a.wheel = append(a.wheel, map[string]any{
		"type":   "scroll",
		"x":      x,
		"y":      y,
		"deltaX": deltaX,
		"deltaY": deltaY,
		"origin": "viewport",
	})
	return a
}

// --- Dispatch ---

// Perform sends the accumulated action sequences to the browser and clears
// the builder so it can be reused.
func (a *Actions) Perform() error {
	actions := []map[string]any{}

	if len(a.pointer) > 0 {
		actions = append(actions, map[string]any{
			"type":       "pointer",
			"id":         "mouse",
			"parameters": map[string]any{"pointerType": "mouse"},
			"actions":    a.pointer,
		})
	}
	if len(a.keys) > 0 {
		actions = append(actions, map[string]any{
			"type":    "key",
			"id":      "keyboard",
			"actions": a.keys,
		})
	}
	if len(a.wheel) > 0 {
		actions = append(actions, map[string]any{
			"type":    "wheel",
			"id":      "wheel",
			"actions": a.wheel,
		})
	}

	_, err := a.sess.t.post(a.sess.path("/actions"), map[string]any{"actions": actions})
	if err != nil {
		return err
	}

	// Reset for reuse.
	a.pointer = nil
	a.keys = nil
	a.wheel = nil
	return nil
}

// Release cancels any currently active actions in the browser.
func (a *Actions) Release() error {
	_, err := a.sess.t.delete(a.sess.path("/actions"))
	return err
}
