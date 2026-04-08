// basic demonstrates fundamental WebDriver operations:
// session creation, navigation, element interaction, and screenshots.
//
// Prerequisites:
//
//	chromedriver --port=9515 &
//
// Run:
//
//	go run ./examples/basic
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/frappuccino2316/woodriver"
)

func main() {
	driverURL := envOr("DRIVER_URL", "http://localhost:9515")

	driver := woodriver.New(driverURL)

	sess, err := driver.NewSession(woodriver.HeadlessChrome())
	if err != nil {
		log.Fatalf("NewSession: %v", err)
	}
	defer sess.Quit()

	// ── Navigation ──────────────────────────────────────────────────────────
	if err := sess.Navigate("https://example.com"); err != nil {
		log.Fatalf("Navigate: %v", err)
	}

	title, err := sess.Title()
	if err != nil {
		log.Fatalf("Title: %v", err)
	}
	fmt.Printf("Title: %s\n", title)

	url, _ := sess.CurrentURL()
	fmt.Printf("URL:   %s\n", url)

	// ── Element interaction ──────────────────────────────────────────────────
	h1, err := sess.FindElement(woodriver.ByCSSSelector, "h1")
	if err != nil {
		log.Fatalf("FindElement h1: %v", err)
	}
	text, _ := h1.Text()
	fmt.Printf("h1:    %s\n", text)

	// ── Explicit wait ────────────────────────────────────────────────────────
	_, err = sess.Wait(5 * time.Second).UntilElement(woodriver.ByCSSSelector, "p")
	if err != nil {
		log.Fatalf("Wait for <p>: %v", err)
	}

	// ── Screenshot ───────────────────────────────────────────────────────────
	png, err := sess.Screenshot()
	if err != nil {
		log.Fatalf("Screenshot: %v", err)
	}
	if err := os.WriteFile("screenshot.png", png, 0o644); err != nil {
		log.Fatalf("WriteFile: %v", err)
	}
	fmt.Printf("Screenshot saved (%d bytes)\n", len(png))

	// ── Window resize ────────────────────────────────────────────────────────
	sess.SetWindowRect(woodriver.Rect{X: 0, Y: 0, Width: 1280, Height: 800})
	rect, _ := sess.WindowRect()
	fmt.Printf("Window: %.0fx%.0f\n", rect.Width, rect.Height)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
