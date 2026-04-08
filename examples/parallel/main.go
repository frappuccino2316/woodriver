// parallel demonstrates SessionPool for concurrent browser automation.
// Multiple URLs are scraped simultaneously, capped by the pool size.
//
// Prerequisites:
//
//	chromedriver --port=9515 &
//
// Run:
//
//	go run ./examples/parallel
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/frappuccino2316/woodriver"
)

func main() {
	driverURL := envOr("DRIVER_URL", "http://localhost:9515")

	urls := []string{
		"https://example.com",
		"https://example.org",
		"https://example.net",
		"https://www.iana.org/domains/reserved",
		"https://www.w3.org",
	}

	const poolSize = 3

	ctx := context.Background()

	// ── セッションプールの作成 ────────────────────────────────────────────────
	pool, err := woodriver.NewSessionPool(ctx,
		woodriver.New(driverURL),
		poolSize,
		woodriver.HeadlessChrome(),
	)
	if err != nil {
		log.Fatalf("NewSessionPool: %v", err)
	}
	defer func() {
		if err := pool.Close(); err != nil {
			log.Printf("pool.Close: %v", err)
		}
	}()

	fmt.Printf("Pool created: %d sessions\n\n", pool.Cap())

	// ── 並列スクレイピング ────────────────────────────────────────────────────
	type result struct {
		url   string
		title string
		err   error
	}

	results := make([]result, len(urls))
	var wg sync.WaitGroup

	for i, u := range urls {
		wg.Add(1)
		go func(idx int, url string) {
			defer wg.Done()

			// タイムアウト付きコンテキストで Acquire
			acquireCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			sess, err := pool.Acquire(acquireCtx)
			if err != nil {
				results[idx] = result{url: url, err: fmt.Errorf("acquire: %w", err)}
				return
			}
			defer pool.Release(sess)

			if err := sess.Navigate(url); err != nil {
				results[idx] = result{url: url, err: fmt.Errorf("navigate: %w", err)}
				return
			}

			title, err := sess.Wait(10 * time.Second).UntilElement(woodriver.ByTagName, "title")
			if err != nil {
				// フォールバック: Title() を直接呼ぶ
				t, e := sess.Title()
				results[idx] = result{url: url, title: t, err: e}
				return
			}

			t, _ := title.Text()
			results[idx] = result{url: url, title: t}
		}(i, u)
	}

	wg.Wait()

	// ── 結果の表示 ────────────────────────────────────────────────────────────
	fmt.Printf("%-45s  %s\n", "URL", "Title")
	fmt.Printf("%-45s  %s\n", "---", "-----")
	for _, r := range results {
		if r.err != nil {
			fmt.Printf("%-45s  ERROR: %v\n", r.url, r.err)
		} else {
			fmt.Printf("%-45s  %s\n", r.url, r.title)
		}
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
