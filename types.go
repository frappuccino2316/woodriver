package woodriver

// By specifies the element location strategy.
type By string

const (
	ByCSSSelector By = "css selector"
	ByXPath       By = "xpath"
	ByLinkText    By = "link text"
	ByPartialLink By = "partial link text"
	ByTagName     By = "tag name"
)

// ByID returns a CSS selector that matches by id attribute.
func ByID(id string) (By, string) { return ByCSSSelector, "#" + id }

// ByName returns a CSS selector that matches by name attribute.
func ByName(name string) (By, string) { return ByCSSSelector, "[name='" + name + "']" }

// Rect represents a bounding rectangle.
type Rect struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}
