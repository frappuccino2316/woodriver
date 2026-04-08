package woodriver

import (
	"encoding/base64"
	"fmt"
	"os"
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

// ============================================================
// Proxy
// ============================================================

// ProxyType specifies the proxy configuration type.
type ProxyType string

const (
	ProxyDirect   ProxyType = "direct"
	ProxyManual   ProxyType = "manual"
	ProxyPAC      ProxyType = "pac"
	ProxyAutodetect ProxyType = "autodetect"
	ProxySystem   ProxyType = "system"
)

// Proxy holds proxy configuration for a browser session.
type Proxy struct {
	Type          ProxyType `json:"proxyType"`
	HTTPProxy     string    `json:"httpProxy,omitempty"`
	HTTPSProxy    string    `json:"sslProxy,omitempty"`
	FTPProxy      string    `json:"ftpProxy,omitempty"`
	SOCKSProxy    string    `json:"socksProxy,omitempty"`
	SOCKSVersion  int       `json:"socksVersion,omitempty"`
	NoProxy       []string  `json:"noProxy,omitempty"`
	PACUrl        string    `json:"proxyAutoconfigUrl,omitempty"`
}

// ManualProxy returns a Proxy configured for manual HTTP/HTTPS proxying.
// host should be in "host:port" form.
func ManualProxy(host string) Proxy {
	return Proxy{
		Type:       ProxyManual,
		HTTPProxy:  host,
		HTTPSProxy: host,
	}
}

// ============================================================
// MobileEmulation
// ============================================================

// MobileDevice holds Chrome mobile emulation settings.
type MobileDevice struct {
	// Use a named device from Chrome DevTools (e.g. "iPhone 12").
	DeviceName string

	// Custom screen dimensions (used when DeviceName is empty).
	Width      int
	Height     int
	PixelRatio float64
	UserAgent  string
	Touch      bool
}

func (m MobileDevice) toMap() map[string]any {
	if m.DeviceName != "" {
		return map[string]any{"deviceName": m.DeviceName}
	}
	metrics := map[string]any{
		"width":             m.Width,
		"height":            m.Height,
		"pixelRatio":        m.PixelRatio,
		"mobile":            true,
		"touch":             m.Touch,
	}
	result := map[string]any{"deviceMetrics": metrics}
	if m.UserAgent != "" {
		result["userAgent"] = m.UserAgent
	}
	return result
}

// ============================================================
// Chrome capabilities
// ============================================================

// ChromeOption is a functional option for Chrome capabilities.
type ChromeOption func(c *chromeConfig)

type chromeConfig struct {
	args            []string
	prefs           map[string]any
	excludeSwitches []string
	extensions      []string // base64-encoded .crx data
	mobileEmulation *MobileDevice
	proxy           *Proxy
	binary          string
	logPrefs        map[string]string
	expOpts         map[string]any
}

// --- Launch args ---

// Headless adds --headless=new.
func Headless() ChromeOption {
	return func(c *chromeConfig) {
		c.args = append(c.args, "--headless=new")
	}
}

// WindowSize sets --window-size.
func WindowSize(width, height int) ChromeOption {
	return func(c *chromeConfig) {
		c.args = append(c.args, fmt.Sprintf("--window-size=%d,%d", width, height))
	}
}

// NoSandbox adds --no-sandbox (required in some container environments).
func NoSandbox() ChromeOption {
	return func(c *chromeConfig) {
		c.args = append(c.args, "--no-sandbox")
	}
}

// DisableGPU adds --disable-gpu (required for headless on Windows).
func DisableGPU() ChromeOption {
	return func(c *chromeConfig) {
		c.args = append(c.args, "--disable-gpu")
	}
}

// DisableDevShmUsage adds --disable-dev-shm-usage (Docker / low-memory envs).
func DisableDevShmUsage() ChromeOption {
	return func(c *chromeConfig) {
		c.args = append(c.args, "--disable-dev-shm-usage")
	}
}

// DisableExtensions adds --disable-extensions.
func DisableExtensions() ChromeOption {
	return func(c *chromeConfig) {
		c.args = append(c.args, "--disable-extensions")
	}
}

// StartMaximized adds --start-maximized.
func StartMaximized() ChromeOption {
	return func(c *chromeConfig) {
		c.args = append(c.args, "--start-maximized")
	}
}

// IgnoreCertificateErrors adds --ignore-certificate-errors.
func IgnoreCertificateErrors() ChromeOption {
	return func(c *chromeConfig) {
		c.args = append(c.args, "--ignore-certificate-errors")
	}
}

// ChromeArg adds an arbitrary Chrome launch argument.
func ChromeArg(arg string) ChromeOption {
	return func(c *chromeConfig) {
		c.args = append(c.args, arg)
	}
}

// ExcludeSwitch removes a default Chrome switch (e.g. "enable-automation").
func ExcludeSwitch(sw string) ChromeOption {
	return func(c *chromeConfig) {
		c.excludeSwitches = append(c.excludeSwitches, sw)
	}
}

// --- Preferences ---

// ChromePref sets a Chrome user preference (written to prefs file).
// Example: ChromePref("download.default_directory", "/tmp")
func ChromePref(key string, value any) ChromeOption {
	return func(c *chromeConfig) {
		if c.prefs == nil {
			c.prefs = map[string]any{}
		}
		c.prefs[key] = value
	}
}

// --- Binary ---

// ChromeBinary sets the path to the Chrome/Chromium executable.
func ChromeBinary(path string) ChromeOption {
	return func(c *chromeConfig) { c.binary = path }
}

// --- Proxy ---

// WithProxy configures a proxy for the Chrome session.
func WithProxy(p Proxy) ChromeOption {
	return func(c *chromeConfig) { c.proxy = &p }
}

// --- Extensions ---

// AddExtension loads a .crx extension file and encodes it for Chrome.
func AddExtension(crxPath string) ChromeOption {
	return func(c *chromeConfig) {
		data, err := os.ReadFile(crxPath)
		if err != nil {
			return // silently skip; caller should validate path first
		}
		c.extensions = append(c.extensions, base64.StdEncoding.EncodeToString(data))
	}
}

// --- Mobile emulation ---

// EmulateDevice enables Chrome's mobile device emulation.
func EmulateDevice(d MobileDevice) ChromeOption {
	return func(c *chromeConfig) { c.mobileEmulation = &d }
}

// --- Logging ---

// LoggingPref sets the log level for a log type.
// logType: "browser", "driver", "performance", etc.
// level:   "OFF", "SEVERE", "WARNING", "INFO", "DEBUG", "ALL"
func LoggingPref(logType, level string) ChromeOption {
	return func(c *chromeConfig) {
		if c.logPrefs == nil {
			c.logPrefs = map[string]string{}
		}
		c.logPrefs[logType] = level
	}
}

// --- Experimental options ---

// ExperimentalOption sets a value in chromeOptions.experimentalOptions.
func ExperimentalOption(key string, value any) ChromeOption {
	return func(c *chromeConfig) {
		if c.expOpts == nil {
			c.expOpts = map[string]any{}
		}
		c.expOpts[key] = value
	}
}

// ChromeCapabilities builds Capabilities for Chrome/Chromium.
func ChromeCapabilities(opts ...ChromeOption) Capabilities {
	cfg := &chromeConfig{}
	for _, o := range opts {
		o(cfg)
	}

	chromeOptions := map[string]any{}
	if len(cfg.args) > 0 {
		chromeOptions["args"] = cfg.args
	}
	if len(cfg.prefs) > 0 {
		chromeOptions["prefs"] = cfg.prefs
	}
	if len(cfg.excludeSwitches) > 0 {
		chromeOptions["excludeSwitches"] = cfg.excludeSwitches
	}
	if len(cfg.extensions) > 0 {
		chromeOptions["extensions"] = cfg.extensions
	}
	if cfg.mobileEmulation != nil {
		chromeOptions["mobileEmulation"] = cfg.mobileEmulation.toMap()
	}
	if cfg.binary != "" {
		chromeOptions["binary"] = cfg.binary
	}
	if len(cfg.expOpts) > 0 {
		for k, v := range cfg.expOpts {
			chromeOptions[k] = v
		}
	}

	caps := Capabilities{
		BrowserName: "chrome",
		Extra:       map[string]any{"goog:chromeOptions": chromeOptions},
	}
	if len(cfg.logPrefs) > 0 {
		caps.Extra["goog:loggingPrefs"] = cfg.logPrefs
	}
	if cfg.proxy != nil {
		caps.Extra["proxy"] = cfg.proxy
	}
	return caps
}

// ============================================================
// Firefox capabilities
// ============================================================

// FirefoxOption is a functional option for Firefox capabilities.
type FirefoxOption func(cfg *firefoxConfig)

type firefoxConfig struct {
	args    []string
	prefs   map[string]any
	binary  string
	env     map[string]string
	profile string // path to an existing Firefox profile directory
}

// FirefoxHeadless adds -headless.
func FirefoxHeadless() FirefoxOption {
	return func(cfg *firefoxConfig) {
		cfg.args = append(cfg.args, "-headless")
	}
}

// FirefoxArg adds an arbitrary Firefox launch argument.
func FirefoxArg(arg string) FirefoxOption {
	return func(cfg *firefoxConfig) {
		cfg.args = append(cfg.args, arg)
	}
}

// FirefoxPref sets a Firefox preference (about:config key).
func FirefoxPref(key string, value any) FirefoxOption {
	return func(cfg *firefoxConfig) {
		if cfg.prefs == nil {
			cfg.prefs = map[string]any{}
		}
		cfg.prefs[key] = value
	}
}

// FirefoxBinary sets the path to the Firefox executable.
func FirefoxBinary(path string) FirefoxOption {
	return func(cfg *firefoxConfig) { cfg.binary = path }
}

// FirefoxProfile sets the path to an existing Firefox profile directory.
func FirefoxProfile(path string) FirefoxOption {
	return func(cfg *firefoxConfig) { cfg.profile = path }
}

// FirefoxEnv sets an environment variable for the Firefox process.
func FirefoxEnv(key, value string) FirefoxOption {
	return func(cfg *firefoxConfig) {
		if cfg.env == nil {
			cfg.env = map[string]string{}
		}
		cfg.env[key] = value
	}
}

// FirefoxWithProxy configures a proxy for the Firefox session.
func FirefoxWithProxy(p Proxy) FirefoxOption {
	return func(cfg *firefoxConfig) {
		if cfg.prefs == nil {
			cfg.prefs = map[string]any{}
		}
		// Firefox proxy via preferences
		cfg.prefs["network.proxy.type"] = 1 // manual
		if p.HTTPProxy != "" {
			cfg.prefs["network.proxy.http"] = p.HTTPProxy
		}
		if p.HTTPSProxy != "" {
			cfg.prefs["network.proxy.ssl"] = p.HTTPSProxy
		}
	}
}

// FirefoxCapabilities builds Capabilities for Firefox.
func FirefoxCapabilities(opts ...FirefoxOption) Capabilities {
	cfg := &firefoxConfig{}
	for _, o := range opts {
		o(cfg)
	}

	firefoxOptions := map[string]any{}
	if len(cfg.args) > 0 {
		firefoxOptions["args"] = cfg.args
	}
	if len(cfg.prefs) > 0 {
		firefoxOptions["prefs"] = cfg.prefs
	}
	if cfg.binary != "" {
		firefoxOptions["binary"] = cfg.binary
	}
	if cfg.profile != "" {
		firefoxOptions["profile"] = cfg.profile
	}
	if len(cfg.env) > 0 {
		firefoxOptions["env"] = cfg.env
	}

	return Capabilities{
		BrowserName: "firefox",
		Extra: map[string]any{
			"moz:firefoxOptions": firefoxOptions,
		},
	}
}

// ============================================================
// Headless convenience preset
// ============================================================

// HeadlessChrome returns capabilities for a headless Chrome session with
// common container-friendly flags pre-applied.
func HeadlessChrome() Capabilities {
	return ChromeCapabilities(
		Headless(),
		NoSandbox(),
		DisableGPU(),
		DisableDevShmUsage(),
		DisableExtensions(),
	)
}

// HeadlessFirefox returns capabilities for a headless Firefox session.
func HeadlessFirefox() Capabilities {
	return FirefoxCapabilities(FirefoxHeadless())
}
