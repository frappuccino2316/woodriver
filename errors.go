package woodriver

import "fmt"

// WebDriverError represents a W3C WebDriver protocol error.
type WebDriverError struct {
	Code    string
	Message string
	Data    any
}

func (e *WebDriverError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("webdriver error [%s]: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("webdriver error [%s]", e.Code)
}

func (e *WebDriverError) Is(target error) bool {
	t, ok := target.(*WebDriverError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// W3C WebDriver error codes.
var (
	ErrElementClickIntercepted   = &WebDriverError{Code: "element click intercepted"}
	ErrElementNotInteractable    = &WebDriverError{Code: "element not interactable"}
	ErrInsecureCertificate       = &WebDriverError{Code: "insecure certificate"}
	ErrInvalidArgument           = &WebDriverError{Code: "invalid argument"}
	ErrInvalidCookieDomain       = &WebDriverError{Code: "invalid cookie domain"}
	ErrInvalidElementState       = &WebDriverError{Code: "invalid element state"}
	ErrInvalidSelector           = &WebDriverError{Code: "invalid selector"}
	ErrInvalidSessionID          = &WebDriverError{Code: "invalid session id"}
	ErrJavaScriptError           = &WebDriverError{Code: "javascript error"}
	ErrMoveTargetOutOfBounds     = &WebDriverError{Code: "move target out of bounds"}
	ErrNoSuchAlert               = &WebDriverError{Code: "no such alert"}
	ErrNoSuchCookie              = &WebDriverError{Code: "no such cookie"}
	ErrNoSuchElement             = &WebDriverError{Code: "no such element"}
	ErrNoSuchFrame               = &WebDriverError{Code: "no such frame"}
	ErrNoSuchWindow              = &WebDriverError{Code: "no such window"}
	ErrScriptTimeout             = &WebDriverError{Code: "script timeout"}
	ErrSessionNotCreated         = &WebDriverError{Code: "session not created"}
	ErrStaleElementReference     = &WebDriverError{Code: "stale element reference"}
	ErrTimeout                   = &WebDriverError{Code: "timeout"}
	ErrUnableToCaptureScreen     = &WebDriverError{Code: "unable to capture screen"}
	ErrUnexpectedAlertOpen       = &WebDriverError{Code: "unexpected alert open"}
	ErrUnknownCommand            = &WebDriverError{Code: "unknown command"}
	ErrUnknownError              = &WebDriverError{Code: "unknown error"}
	ErrUnknownMethod             = &WebDriverError{Code: "unknown method"}
	ErrUnsupportedOperation      = &WebDriverError{Code: "unsupported operation"}
)
