package gendry

import "fmt"
import "net/url"
import "net/http"

const (
	defaultShieldStyle   = "flat-square"
	shieldConfigTemplate = "%s-%.2f%%-%s"
	shieldURLTemplate    = "https://img.shields.io/badge/%s.svg"
)

// NewDisplayAPI returns a new APIEndpoint capable of returning badge data from shields.io.
func NewDisplayAPI() APIEndpoint {
	return &displayAPI{}
}

// displayAPI is responsible for writing the svg badge result from shields.io given a report name.
type displayAPI struct {
	notImplementedRoute
}

func (a *displayAPI) Get(writer http.ResponseWriter, request *http.Request, params url.Values) {
	fmt.Fprintf(writer, "project[%s]\n", params.Get("project"))
	fmt.Fprintf(writer, "tag[%s]\n", params.Get("tag"))
	fmt.Fprintf(writer, "format[%s]\n", params.Get("format"))
}
