// scraping demonstrates practical scraping patterns:
// waiting for dynamic content, extracting multiple elements,
// JavaScript execution, and cookie manipulation.
//
// Prerequisites:
//
//	chromedriver --port=9515 &
//
// Run:
//
//	go run ./examples/scraping
package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/frappuccino2316/woodriver"
)

func main() {
	driverURL := envOr("DRIVER_URL", "http://localhost:9515")

	sess, err := woodriver.New(driverURL).NewSession(woodriver.ChromeCapabilities(
		woodriver.Headless(),
		woodriver.NoSandbox(),
		woodriver.DisableGPU(),
		woodriver.WindowSize(1280, 900),
	))
	if err != nil {
		log.Fatalf("NewSession: %v", err)
	}
	defer sess.Quit()

	if err := sess.Navigate("https://example.com"); err != nil {
		log.Fatalf("Navigate: %v", err)
	}

	// ── 明示的待機: タイトルが確定するまで待つ ──────────────────────────────
	if err := sess.Wait(10 * time.Second).Until(woodriver.TitleContains("Example")); err != nil {
		log.Fatalf("Wait for title: %v", err)
	}

	// ── 複数要素の一括取得 ───────────────────────────────────────────────────
	paragraphs, err := sess.FindElements(woodriver.ByCSSSelector, "p")
	if err != nil {
		log.Fatalf("FindElements: %v", err)
	}
	fmt.Printf("Found %d paragraphs\n", len(paragraphs))
	for i, p := range paragraphs {
		text, _ := p.Text()
		fmt.Printf("  [%d] %s\n", i, text)
	}

	// ── JavaScript でページ情報を取得 ────────────────────────────────────────
	result, err := sess.Execute(`return {
		title:    document.title,
		url:      location.href,
		width:    window.innerWidth,
		height:   window.innerHeight,
	}`)
	if err != nil {
		log.Fatalf("Execute: %v", err)
	}
	if m, ok := result.(map[string]any); ok {
		fmt.Printf("JS result: title=%v, width=%v\n", m["title"], m["width"])
	}

	// ── Cookie の追加・取得 ──────────────────────────────────────────────────
	if err := sess.AddCookie(woodriver.Cookie{
		Name:  "scraper",
		Value: "woodriver",
		Path:  "/",
	}); err != nil {
		log.Fatalf("AddCookie: %v", err)
	}

	c, err := sess.Cookie("scraper")
	if err != nil {
		log.Fatalf("Cookie: %v", err)
	}
	fmt.Printf("Cookie: %s=%s\n", c.Name, c.Value)

	// ── エラーハンドリング: 存在しない要素 ──────────────────────────────────
	_, err = sess.FindElement(woodriver.ByCSSSelector, "#does-not-exist")
	if errors.Is(err, woodriver.ErrNoSuchElement) {
		fmt.Println("Element not found (expected)")
	} else if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
