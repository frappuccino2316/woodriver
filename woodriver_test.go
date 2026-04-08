package woodriver_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/frappuccino2316/woodriver"
)

// newMockServer creates a test WebDriver server that responds to session creation
// and a configurable set of endpoint handlers.
func newMockServer(t *testing.T, handlers map[string]http.HandlerFunc) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	// Default session creation endpoint.
	mux.HandleFunc("POST /session", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"value": map[string]any{"sessionId": "test-session-id"},
		})
	})

	for pattern, h := range handlers {
		mux.HandleFunc(pattern, h)
	}

	return httptest.NewServer(mux)
}

func respond(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"value": value})
}

func TestNewSession(t *testing.T) {
	srv := newMockServer(t, nil)
	defer srv.Close()

	driver := woodriver.New(srv.URL)
	sess, err := driver.NewSession(woodriver.ChromeCapabilities())
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if sess == nil {
		t.Fatal("expected non-nil session")
	}
	_ = sess.Quit()
}

func TestNavigateAndCurrentURL(t *testing.T) {
	const wantURL = "https://example.com"

	srv := newMockServer(t, map[string]http.HandlerFunc{
		"POST /session/test-session-id/url": func(w http.ResponseWriter, r *http.Request) {
			respond(w, nil)
		},
		"GET /session/test-session-id/url": func(w http.ResponseWriter, r *http.Request) {
			respond(w, wantURL)
		},
		"DELETE /session/test-session-id": func(w http.ResponseWriter, r *http.Request) {
			respond(w, nil)
		},
	})
	defer srv.Close()

	sess, _ := woodriver.New(srv.URL).NewSession(woodriver.ChromeCapabilities())
	defer sess.Quit()

	if err := sess.Navigate(wantURL); err != nil {
		t.Fatalf("Navigate: %v", err)
	}
	got, err := sess.CurrentURL()
	if err != nil {
		t.Fatalf("CurrentURL: %v", err)
	}
	if got != wantURL {
		t.Errorf("CurrentURL = %q, want %q", got, wantURL)
	}
}

func TestTitle(t *testing.T) {
	const wantTitle = "Example Domain"

	srv := newMockServer(t, map[string]http.HandlerFunc{
		"GET /session/test-session-id/title": func(w http.ResponseWriter, r *http.Request) {
			respond(w, wantTitle)
		},
		"DELETE /session/test-session-id": func(w http.ResponseWriter, r *http.Request) {
			respond(w, nil)
		},
	})
	defer srv.Close()

	sess, _ := woodriver.New(srv.URL).NewSession(woodriver.ChromeCapabilities())
	defer sess.Quit()

	got, err := sess.Title()
	if err != nil {
		t.Fatalf("Title: %v", err)
	}
	if got != wantTitle {
		t.Errorf("Title = %q, want %q", got, wantTitle)
	}
}

func TestFindElement(t *testing.T) {
	const elemKey = "element-6066-11e4-a52e-4f735466cecf"
	const elemID = "abc-123"

	srv := newMockServer(t, map[string]http.HandlerFunc{
		"POST /session/test-session-id/element": func(w http.ResponseWriter, r *http.Request) {
			respond(w, map[string]string{elemKey: elemID})
		},
		"GET /session/test-session-id/element/abc-123/text": func(w http.ResponseWriter, r *http.Request) {
			respond(w, "Hello World")
		},
		"DELETE /session/test-session-id": func(w http.ResponseWriter, r *http.Request) {
			respond(w, nil)
		},
	})
	defer srv.Close()

	sess, _ := woodriver.New(srv.URL).NewSession(woodriver.ChromeCapabilities())
	defer sess.Quit()

	el, err := sess.FindElement(woodriver.ByCSSSelector, "h1")
	if err != nil {
		t.Fatalf("FindElement: %v", err)
	}
	text, err := el.Text()
	if err != nil {
		t.Fatalf("Text: %v", err)
	}
	if text != "Hello World" {
		t.Errorf("Text = %q, want %q", text, "Hello World")
	}
}

func TestNoSuchElementError(t *testing.T) {
	srv := newMockServer(t, map[string]http.HandlerFunc{
		"POST /session/test-session-id/element": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			respond(w, map[string]any{
				"error":   "no such element",
				"message": "element not found",
			})
		},
		"DELETE /session/test-session-id": func(w http.ResponseWriter, r *http.Request) {
			respond(w, nil)
		},
	})
	defer srv.Close()

	sess, _ := woodriver.New(srv.URL).NewSession(woodriver.ChromeCapabilities())
	defer sess.Quit()

	_, err := sess.FindElement(woodriver.ByCSSSelector, "#nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var wdErr *woodriver.WebDriverError
	if !isWebDriverError(err, &wdErr) || wdErr.Code != "no such element" {
		t.Errorf("expected no such element error, got: %v", err)
	}
}

func TestWaiterTimeout(t *testing.T) {
	srv := newMockServer(t, map[string]http.HandlerFunc{
		"POST /session/test-session-id/element": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			respond(w, map[string]any{"error": "no such element", "message": ""})
		},
		"DELETE /session/test-session-id": func(w http.ResponseWriter, r *http.Request) {
			respond(w, nil)
		},
	})
	defer srv.Close()

	sess, _ := woodriver.New(srv.URL).NewSession(woodriver.ChromeCapabilities())
	defer sess.Quit()

	start := time.Now()
	_, err := sess.Wait(300 * time.Millisecond).UntilElement(woodriver.ByCSSSelector, "#missing")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error")
	}
	if elapsed < 200*time.Millisecond {
		t.Errorf("returned too quickly: %v", elapsed)
	}
}

func TestScreenshot(t *testing.T) {
	// 1x1 white PNG base64
	const encoded = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI6QAAAABJRU5ErkJggg=="

	srv := newMockServer(t, map[string]http.HandlerFunc{
		"GET /session/test-session-id/screenshot": func(w http.ResponseWriter, r *http.Request) {
			respond(w, encoded)
		},
		"DELETE /session/test-session-id": func(w http.ResponseWriter, r *http.Request) {
			respond(w, nil)
		},
	})
	defer srv.Close()

	sess, _ := woodriver.New(srv.URL).NewSession(woodriver.ChromeCapabilities())
	defer sess.Quit()

	data, err := sess.Screenshot()
	if err != nil {
		t.Fatalf("Screenshot: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty screenshot data")
	}
}

// --- Phase 2: Actions API ---

func TestActionsMouseClick(t *testing.T) {
	var gotBody map[string]any

	srv := newMockServer(t, map[string]http.HandlerFunc{
		"POST /session/test-session-id/actions": func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&gotBody)
			respond(w, nil)
		},
		"DELETE /session/test-session-id": func(w http.ResponseWriter, r *http.Request) {
			respond(w, nil)
		},
	})
	defer srv.Close()

	sess, _ := woodriver.New(srv.URL).NewSession(woodriver.ChromeCapabilities())
	defer sess.Quit()

	err := sess.Actions().
		MouseMove(100, 200).
		MouseClick(woodriver.MouseLeft).
		Perform()
	if err != nil {
		t.Fatalf("Actions.Perform: %v", err)
	}

	acts, _ := gotBody["actions"].([]any)
	if len(acts) == 0 {
		t.Fatal("expected actions in payload")
	}
	first := acts[0].(map[string]any)
	if first["type"] != "pointer" {
		t.Errorf("expected pointer action, got %v", first["type"])
	}
}

func TestActionsKeyboard(t *testing.T) {
	var gotBody map[string]any

	srv := newMockServer(t, map[string]http.HandlerFunc{
		"POST /session/test-session-id/actions": func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&gotBody)
			respond(w, nil)
		},
		"DELETE /session/test-session-id": func(w http.ResponseWriter, r *http.Request) {
			respond(w, nil)
		},
	})
	defer srv.Close()

	sess, _ := woodriver.New(srv.URL).NewSession(woodriver.ChromeCapabilities())
	defer sess.Quit()

	err := sess.Actions().
		KeyDown(woodriver.KeyControl).
		KeySendKeys("a").
		KeyUp(woodriver.KeyControl).
		Perform()
	if err != nil {
		t.Fatalf("Actions keyboard: %v", err)
	}

	acts, _ := gotBody["actions"].([]any)
	if len(acts) == 0 {
		t.Fatal("expected actions in payload")
	}
	keyAction := acts[0].(map[string]any)
	if keyAction["type"] != "key" {
		t.Errorf("expected key action, got %v", keyAction["type"])
	}
}

func TestActionsScroll(t *testing.T) {
	var gotBody map[string]any

	srv := newMockServer(t, map[string]http.HandlerFunc{
		"POST /session/test-session-id/actions": func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&gotBody)
			respond(w, nil)
		},
		"DELETE /session/test-session-id": func(w http.ResponseWriter, r *http.Request) {
			respond(w, nil)
		},
	})
	defer srv.Close()

	sess, _ := woodriver.New(srv.URL).NewSession(woodriver.ChromeCapabilities())
	defer sess.Quit()

	err := sess.Actions().Scroll(0, 0, 0, 300).Perform()
	if err != nil {
		t.Fatalf("Actions.Scroll: %v", err)
	}

	acts, _ := gotBody["actions"].([]any)
	if len(acts) == 0 {
		t.Fatal("expected actions in payload")
	}
	wheelAction := acts[0].(map[string]any)
	if wheelAction["type"] != "wheel" {
		t.Errorf("expected wheel action, got %v", wheelAction["type"])
	}
}

// --- Phase 2: Window operations ---

func TestWindowHandles(t *testing.T) {
	srv := newMockServer(t, map[string]http.HandlerFunc{
		"GET /session/test-session-id/window/handles": func(w http.ResponseWriter, r *http.Request) {
			respond(w, []string{"handle-1", "handle-2"})
		},
		"DELETE /session/test-session-id": func(w http.ResponseWriter, r *http.Request) {
			respond(w, nil)
		},
	})
	defer srv.Close()

	sess, _ := woodriver.New(srv.URL).NewSession(woodriver.ChromeCapabilities())
	defer sess.Quit()

	handles, err := sess.WindowHandles()
	if err != nil {
		t.Fatalf("WindowHandles: %v", err)
	}
	if len(handles) != 2 {
		t.Errorf("len(handles) = %d, want 2", len(handles))
	}
}

func TestNewWindow(t *testing.T) {
	srv := newMockServer(t, map[string]http.HandlerFunc{
		"POST /session/test-session-id/window/new": func(w http.ResponseWriter, r *http.Request) {
			respond(w, map[string]any{"handle": "new-handle", "type": "tab"})
		},
		"DELETE /session/test-session-id": func(w http.ResponseWriter, r *http.Request) {
			respond(w, nil)
		},
	})
	defer srv.Close()

	sess, _ := woodriver.New(srv.URL).NewSession(woodriver.ChromeCapabilities())
	defer sess.Quit()

	wh, err := sess.NewWindow(woodriver.WindowTypeTab)
	if err != nil {
		t.Fatalf("NewWindow: %v", err)
	}
	if wh.Handle != "new-handle" {
		t.Errorf("handle = %q, want %q", wh.Handle, "new-handle")
	}
	if wh.Type != "tab" {
		t.Errorf("type = %q, want %q", wh.Type, "tab")
	}
}

func TestMaximize(t *testing.T) {
	called := false
	srv := newMockServer(t, map[string]http.HandlerFunc{
		"POST /session/test-session-id/window/maximize": func(w http.ResponseWriter, r *http.Request) {
			called = true
			respond(w, map[string]any{"x": 0, "y": 0, "width": 1920, "height": 1080})
		},
		"DELETE /session/test-session-id": func(w http.ResponseWriter, r *http.Request) {
			respond(w, nil)
		},
	})
	defer srv.Close()

	sess, _ := woodriver.New(srv.URL).NewSession(woodriver.ChromeCapabilities())
	defer sess.Quit()

	if err := sess.Maximize(); err != nil {
		t.Fatalf("Maximize: %v", err)
	}
	if !called {
		t.Error("maximize endpoint was not called")
	}
}

func TestSwitchToFrame(t *testing.T) {
	var gotBody map[string]any

	srv := newMockServer(t, map[string]http.HandlerFunc{
		"POST /session/test-session-id/frame": func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&gotBody)
			respond(w, nil)
		},
		"DELETE /session/test-session-id": func(w http.ResponseWriter, r *http.Request) {
			respond(w, nil)
		},
	})
	defer srv.Close()

	sess, _ := woodriver.New(srv.URL).NewSession(woodriver.ChromeCapabilities())
	defer sess.Quit()

	if err := sess.SwitchToFrame(0); err != nil {
		t.Fatalf("SwitchToFrame: %v", err)
	}
	if gotBody["id"].(float64) != 0 {
		t.Errorf("frame id = %v, want 0", gotBody["id"])
	}
}

func TestAlertAcceptDismiss(t *testing.T) {
	acceptCalled, dismissCalled := false, false

	srv := newMockServer(t, map[string]http.HandlerFunc{
		"GET /session/test-session-id/alert/text": func(w http.ResponseWriter, r *http.Request) {
			respond(w, "Are you sure?")
		},
		"POST /session/test-session-id/alert/accept": func(w http.ResponseWriter, r *http.Request) {
			acceptCalled = true
			respond(w, nil)
		},
		"POST /session/test-session-id/alert/dismiss": func(w http.ResponseWriter, r *http.Request) {
			dismissCalled = true
			respond(w, nil)
		},
		"DELETE /session/test-session-id": func(w http.ResponseWriter, r *http.Request) {
			respond(w, nil)
		},
	})
	defer srv.Close()

	sess, _ := woodriver.New(srv.URL).NewSession(woodriver.ChromeCapabilities())
	defer sess.Quit()

	text, err := sess.AlertText()
	if err != nil {
		t.Fatalf("AlertText: %v", err)
	}
	if text != "Are you sure?" {
		t.Errorf("AlertText = %q, want %q", text, "Are you sure?")
	}

	if err := sess.AcceptAlert(); err != nil {
		t.Fatalf("AcceptAlert: %v", err)
	}
	if err := sess.DismissAlert(); err != nil {
		t.Fatalf("DismissAlert: %v", err)
	}

	if !acceptCalled {
		t.Error("accept endpoint not called")
	}
	if !dismissCalled {
		t.Error("dismiss endpoint not called")
	}
}

// --- Phase 3: Capabilities ---

func TestChromeCapabilitiesHeadlessPreset(t *testing.T) {
	caps := woodriver.HeadlessChrome()
	if caps.BrowserName != "chrome" {
		t.Errorf("BrowserName = %q, want chrome", caps.BrowserName)
	}
	opts, ok := caps.Extra["goog:chromeOptions"].(map[string]any)
	if !ok {
		t.Fatal("goog:chromeOptions missing")
	}
	args, _ := opts["args"].([]string)
	hasHeadless := false
	hasNoSandbox := false
	for _, a := range args {
		if a == "--headless=new" {
			hasHeadless = true
		}
		if a == "--no-sandbox" {
			hasNoSandbox = true
		}
	}
	if !hasHeadless {
		t.Error("expected --headless=new in args")
	}
	if !hasNoSandbox {
		t.Error("expected --no-sandbox in args")
	}
}

func TestChromeCapabilitiesWithProxy(t *testing.T) {
	caps := woodriver.ChromeCapabilities(
		woodriver.WithProxy(woodriver.ManualProxy("proxy.example.com:8080")),
	)
	proxy, ok := caps.Extra["proxy"].(*woodriver.Proxy)
	if !ok || proxy == nil {
		t.Fatal("proxy not set in capabilities")
	}
	if proxy.HTTPProxy != "proxy.example.com:8080" {
		t.Errorf("HTTPProxy = %q", proxy.HTTPProxy)
	}
}

func TestChromeCapabilitiesWithMobileEmulation(t *testing.T) {
	caps := woodriver.ChromeCapabilities(
		woodriver.EmulateDevice(woodriver.MobileDevice{DeviceName: "iPhone 12"}),
	)
	opts := caps.Extra["goog:chromeOptions"].(map[string]any)
	em, ok := opts["mobileEmulation"].(map[string]any)
	if !ok {
		t.Fatal("mobileEmulation missing")
	}
	if em["deviceName"] != "iPhone 12" {
		t.Errorf("deviceName = %v", em["deviceName"])
	}
}

func TestFirefoxCapabilitiesWithPrefs(t *testing.T) {
	caps := woodriver.FirefoxCapabilities(
		woodriver.FirefoxHeadless(),
		woodriver.FirefoxPref("browser.startup.homepage", "about:blank"),
	)
	opts := caps.Extra["moz:firefoxOptions"].(map[string]any)
	prefs, ok := opts["prefs"].(map[string]any)
	if !ok {
		t.Fatal("prefs missing")
	}
	if prefs["browser.startup.homepage"] != "about:blank" {
		t.Errorf("pref value = %v", prefs["browser.startup.homepage"])
	}
}

// --- Phase 3: Cookies ---

func TestCookies(t *testing.T) {
	expiry := int64(9999999999)
	srv := newMockServer(t, map[string]http.HandlerFunc{
		"GET /session/test-session-id/cookie": func(w http.ResponseWriter, r *http.Request) {
			respond(w, []map[string]any{
				{"name": "session", "value": "abc123", "expiry": expiry},
				{"name": "lang", "value": "ja"},
			})
		},
		"DELETE /session/test-session-id": func(w http.ResponseWriter, r *http.Request) {
			respond(w, nil)
		},
	})
	defer srv.Close()

	sess, _ := woodriver.New(srv.URL).NewSession(woodriver.ChromeCapabilities())
	defer sess.Quit()

	cookies, err := sess.Cookies()
	if err != nil {
		t.Fatalf("Cookies: %v", err)
	}
	if len(cookies) != 2 {
		t.Fatalf("len(cookies) = %d, want 2", len(cookies))
	}
	if cookies[0].Name != "session" || cookies[0].Value != "abc123" {
		t.Errorf("cookie[0] = %+v", cookies[0])
	}
}

func TestAddAndDeleteCookie(t *testing.T) {
	addCalled, deleteCalled := false, false
	var addedCookie map[string]any

	srv := newMockServer(t, map[string]http.HandlerFunc{
		"POST /session/test-session-id/cookie": func(w http.ResponseWriter, r *http.Request) {
			addCalled = true
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			addedCookie = body["cookie"].(map[string]any)
			respond(w, nil)
		},
		"DELETE /session/test-session-id/cookie/mycookie": func(w http.ResponseWriter, r *http.Request) {
			deleteCalled = true
			respond(w, nil)
		},
		"DELETE /session/test-session-id": func(w http.ResponseWriter, r *http.Request) {
			respond(w, nil)
		},
	})
	defer srv.Close()

	sess, _ := woodriver.New(srv.URL).NewSession(woodriver.ChromeCapabilities())
	defer sess.Quit()

	if err := sess.AddCookie(woodriver.Cookie{Name: "mycookie", Value: "myvalue", Secure: true}); err != nil {
		t.Fatalf("AddCookie: %v", err)
	}
	if !addCalled {
		t.Error("add cookie endpoint not called")
	}
	if addedCookie["name"] != "mycookie" {
		t.Errorf("cookie name = %v", addedCookie["name"])
	}

	if err := sess.DeleteCookie("mycookie"); err != nil {
		t.Fatalf("DeleteCookie: %v", err)
	}
	if !deleteCalled {
		t.Error("delete cookie endpoint not called")
	}
}

// --- Phase 3: SessionPool ---

func TestSessionPoolAcquireRelease(t *testing.T) {
	const poolSize = 3
	srv := newMockServer(t, map[string]http.HandlerFunc{
		"DELETE /session/test-session-id": func(w http.ResponseWriter, r *http.Request) {
			respond(w, nil)
		},
	})
	defer srv.Close()

	pool, err := woodriver.NewSessionPool(context.Background(), woodriver.New(srv.URL), poolSize, woodriver.ChromeCapabilities())
	if err != nil {
		t.Fatalf("NewSessionPool: %v", err)
	}

	if pool.Cap() != poolSize {
		t.Errorf("Cap = %d, want %d", pool.Cap(), poolSize)
	}

	// Acquire all sessions.
	sessions := make([]woodriver.WindowOps, poolSize)
	for i := range sessions {
		s, err := pool.Acquire(context.Background())
		if err != nil {
			t.Fatalf("Acquire[%d]: %v", i, err)
		}
		sessions[i] = s
	}

	if pool.Len() != 0 {
		t.Errorf("Len = %d after acquiring all, want 0", pool.Len())
	}

	// Release all.
	for _, s := range sessions {
		pool.Release(s)
	}

	if pool.Len() != poolSize {
		t.Errorf("Len = %d after releasing all, want %d", pool.Len(), poolSize)
	}

	if err := pool.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestSessionPoolConcurrent(t *testing.T) {
	const poolSize = 3
	const workers = 9

	var peak int64

	srv := newMockServer(t, map[string]http.HandlerFunc{
		"GET /session/test-session-id/title": func(w http.ResponseWriter, r *http.Request) {
			respond(w, "Test")
		},
		"DELETE /session/test-session-id": func(w http.ResponseWriter, r *http.Request) {
			respond(w, nil)
		},
	})
	defer srv.Close()

	pool, err := woodriver.NewSessionPool(context.Background(), woodriver.New(srv.URL), poolSize, woodriver.ChromeCapabilities())
	if err != nil {
		t.Fatalf("NewSessionPool: %v", err)
	}
	defer pool.Close()

	var wg sync.WaitGroup
	var inUse int64

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sess, err := pool.Acquire(context.Background())
			if err != nil {
				t.Errorf("Acquire: %v", err)
				return
			}
			cur := atomic.AddInt64(&inUse, 1)
			if cur > peak {
				atomic.StoreInt64(&peak, cur)
			}
			// Simulate work.
			sess.Title()
			atomic.AddInt64(&inUse, -1)
			pool.Release(sess)
		}()
	}
	wg.Wait()

	if peak > int64(poolSize) {
		t.Errorf("peak concurrent sessions = %d, exceeds pool size %d", peak, poolSize)
	}
}

func TestSessionPoolContextCancel(t *testing.T) {
	srv := newMockServer(t, map[string]http.HandlerFunc{
		"DELETE /session/test-session-id": func(w http.ResponseWriter, r *http.Request) {
			respond(w, nil)
		},
	})
	defer srv.Close()

	// Pool of 1 — acquire it, then try to acquire again with a cancelled context.
	pool, _ := woodriver.NewSessionPool(context.Background(), woodriver.New(srv.URL), 1, woodriver.ChromeCapabilities())
	defer pool.Close()

	sess, _ := pool.Acquire(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := pool.Acquire(ctx)
	if err == nil {
		t.Fatal("expected context error when pool is exhausted")
	}

	pool.Release(sess)
}

// isWebDriverError uses errors.As semantics.
func isWebDriverError(err error, target **woodriver.WebDriverError) bool {
	if err == nil {
		return false
	}
	if wdErr, ok := err.(*woodriver.WebDriverError); ok {
		if target != nil {
			*target = wdErr
		}
		return true
	}
	return false
}
