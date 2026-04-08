package woodriver

import (
	"fmt"
	"time"
)

// Capabilities holds browser configuration for session creation.
type Capabilities struct {
	BrowserName    string
	BrowserVersion string
	PlatformName   string
	Timeouts       Timeouts
	Extra          map[string]any
}

// Timeouts configures script, page load, and implicit wait durations.
type Timeouts struct {
	Script   time.Duration
	PageLoad time.Duration
	Implicit time.Duration
}

func (t Timeouts) toMap() map[string]int64 {
	m := map[string]int64{}
	if t.Script > 0 {
		m["script"] = t.Script.Milliseconds()
	}
	if t.PageLoad > 0 {
		m["pageLoad"] = t.PageLoad.Milliseconds()
	}
	if t.Implicit > 0 {
		m["implicit"] = t.Implicit.Milliseconds()
	}
	return m
}

// toW3C converts Capabilities to the W3C JSON payload.
func (c Capabilities) toW3C() map[string]any {
	alwaysMatch := map[string]any{}
	if c.BrowserName != "" {
		alwaysMatch["browserName"] = c.BrowserName
	}
	if c.BrowserVersion != "" {
		alwaysMatch["browserVersion"] = c.BrowserVersion
	}
	if c.PlatformName != "" {
		alwaysMatch["platformName"] = c.PlatformName
	}
	if m := c.Timeouts.toMap(); len(m) > 0 {
		alwaysMatch["timeouts"] = m
	}
	for k, v := range c.Extra {
		alwaysMatch[k] = v
	}
	return map[string]any{
		"capabilities": map[string]any{
			"alwaysMatch": alwaysMatch,
		},
	}
}

// ChromeOption is a functional option for Chrome capabilities.
type ChromeOption func(args *chromeArgs)

type chromeArgs struct {
	args        []string
	prefs       map[string]any
	excludeSwitches []string
}

// Headless adds the --headless flag.
func Headless() ChromeOption {
	return func(a *chromeArgs) {
		a.args = append(a.args, "--headless=new")
	}
}

// WindowSize sets the browser window size.
func WindowSize(width, height int) ChromeOption {
	return func(a *chromeArgs) {
		a.args = append(a.args, fmt.Sprintf("--window-size=%d,%d", width, height))
	}
}

// ChromeArg adds a raw Chrome argument.
func ChromeArg(arg string) ChromeOption {
	return func(a *chromeArgs) {
		a.args = append(a.args, arg)
	}
}

// ChromeCapabilities builds Capabilities for Chrome/Chromium.
func ChromeCapabilities(opts ...ChromeOption) Capabilities {
	args := &chromeArgs{}
	for _, o := range opts {
		o(args)
	}

	chromeOptions := map[string]any{}
	if len(args.args) > 0 {
		chromeOptions["args"] = args.args
	}
	if len(args.prefs) > 0 {
		chromeOptions["prefs"] = args.prefs
	}

	return Capabilities{
		BrowserName: "chrome",
		Extra: map[string]any{
			"goog:chromeOptions": chromeOptions,
		},
	}
}

// FirefoxOption is a functional option for Firefox capabilities.
type FirefoxOption func(prefs map[string]any, args *[]string)

// FirefoxCapabilities builds Capabilities for Firefox.
func FirefoxCapabilities(opts ...FirefoxOption) Capabilities {
	prefs := map[string]any{}
	args := []string{}
	for _, o := range opts {
		o(prefs, &args)
	}

	firefoxOptions := map[string]any{}
	if len(prefs) > 0 {
		firefoxOptions["prefs"] = prefs
	}
	if len(args) > 0 {
		firefoxOptions["args"] = args
	}

	return Capabilities{
		BrowserName: "firefox",
		Extra: map[string]any{
			"moz:firefoxOptions": firefoxOptions,
		},
	}
}

// FirefoxHeadless adds -headless to Firefox.
func FirefoxHeadless() FirefoxOption {
	return func(_ map[string]any, args *[]string) {
		*args = append(*args, "-headless")
	}
}
