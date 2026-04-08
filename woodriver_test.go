package woodriver_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
