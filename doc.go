// Package woodriver is a Go client for the W3C WebDriver protocol.
//
// It lets you automate web browsers (Chrome, Firefox, etc.) through their
// respective WebDriver servers (ChromeDriver, GeckoDriver).
//
// # Quick start
//
//	driver := woodriver.New("http://localhost:9515")
//
//	sess, err := driver.NewSession(woodriver.HeadlessChrome())
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer sess.Quit()
//
//	if err := sess.Navigate("https://example.com"); err != nil {
//	    log.Fatal(err)
//	}
//
//	title, _ := sess.Title()
//	fmt.Println(title) // "Example Domain"
//
// # Interfaces
//
// The package exposes two interfaces that callers use:
//
//   - [Session]    – navigation, element search, JavaScript, screenshots.
//   - [WindowOps]  – extends Session with window/tab management, frame
//     switching, alert handling, cookies, and the Actions API.
//
// [Driver.NewSession] returns a [WindowOps], so callers have access to the
// full API without any type assertions.
//
// # Finding elements
//
//	el, err := sess.FindElement(woodriver.ByCSSSelector, "h1")
//	text, err := el.Text()
//
// Helper constructors [ByID] and [ByName] generate CSS selectors:
//
//	by, value := woodriver.ByID("submit-btn")
//	el, err := sess.FindElement(by, value)
//
// # Explicit waits
//
//	el, err := sess.Wait(10 * time.Second).UntilElement(woodriver.ByCSSSelector, ".result")
//
//	err = sess.Wait(5 * time.Second).Until(woodriver.TitleContains("Dashboard"))
//
// # Actions API
//
//	err := sess.Actions().
//	    MouseMoveToElement(el).
//	    MouseClick(woodriver.MouseLeft).
//	    KeyDown(woodriver.KeyControl).
//	    KeySendKeys("c").
//	    KeyUp(woodriver.KeyControl).
//	    Perform()
//
// # Parallel sessions
//
//	pool, err := woodriver.NewSessionPool(ctx, driver, 4, woodriver.HeadlessChrome())
//	defer pool.Close()
//
//	sess, err := pool.Acquire(ctx)
//	defer pool.Release(sess)
//
// # Architecture
//
// HTTP communication with the WebDriver server is handled by the internal
// transport package (internal/transport) and is not part of the public API.
// Callers interact only with the types and functions exported by this package.
package woodriver
