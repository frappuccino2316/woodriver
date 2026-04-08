// actions demonstrates the W3C Actions API:
// mouse movement, clicks, keyboard shortcuts, and wheel scrolling.
//
// Prerequisites:
//
//	chromedriver --port=9515 &
//
// Run:
//
//	go run ./examples/actions
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

	sess, err := woodriver.New(driverURL).NewSession(woodriver.HeadlessChrome())
	if err != nil {
		log.Fatalf("NewSession: %v", err)
	}
	defer sess.Quit()

	if err := sess.Navigate("https://example.com"); err != nil {
		log.Fatalf("Navigate: %v", err)
	}

	// ── マウス: 要素へ移動してクリック ───────────────────────────────────────
	link, err := sess.Wait(5 * time.Second).UntilElement(woodriver.ByCSSSelector, "a")
	if err != nil {
		log.Fatalf("Wait for link: %v", err)
	}

	if err := sess.Actions().ClickElement(link).Perform(); err != nil {
		log.Fatalf("ClickElement: %v", err)
	}
	fmt.Println("Clicked link")

	sess.Back()

	// ── キーボード: Ctrl+A で全選択、その後 Escape ───────────────────────────
	if err := sess.Actions().
		KeyDown(woodriver.KeyControl).
		KeySendKeys("a").
		KeyUp(woodriver.KeyControl).
		KeyDown(woodriver.KeyEscape).
		KeyUp(woodriver.KeyEscape).
		Perform(); err != nil {
		log.Fatalf("Keyboard actions: %v", err)
	}
	fmt.Println("Keyboard shortcut performed")

	// ── ホイール: 300px 下スクロール ─────────────────────────────────────────
	if err := sess.Actions().Scroll(0, 0, 0, 300).Perform(); err != nil {
		log.Fatalf("Scroll: %v", err)
	}
	fmt.Println("Scrolled down 300px")

	// ── マウス: ダブルクリック ───────────────────────────────────────────────
	if err := sess.Actions().
		MouseMove(400, 300).
		MouseDoubleClick(woodriver.MouseLeft).
		Perform(); err != nil {
		log.Fatalf("DoubleClick: %v", err)
	}
	fmt.Println("Double-clicked at (400, 300)")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
