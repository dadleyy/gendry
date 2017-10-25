package gendry

import "io"
import "log"
import "fmt"
import "net/url"
import "net/http"

const (
	defaultShieldStyle   = "flat-square"
	shieldConfigTemplate = "%s-%.2f%%-%s"
	shieldURLTemplate    = "https://img.shields.io/badge/%s.svg"
)

// NewBadgeAPI returns a new APIEndpoint capable of returning badge data from shields.io.
func NewBadgeAPI(store ReportStore) APIEndpoint {
	return &badgeAPI{store: store}
}

// BadgeAPI is resposnible for writing the svg badge result from shields.io given a report name.
type badgeAPI struct {
	notImplementedRoute
	store ReportStore
}

// Get response to http GET methods for the BadgeAPI endpoint.
func (api *badgeAPI) Get(responseWriter http.ResponseWriter, request *http.Request, params url.Values) {
	log.Printf("matched badge route, params: %v", params)
	shieldText := "generated--coverage"

	if text := request.URL.Query().Get("text"); text != "" {
		shieldText = text
	}

	_, txt, e := api.store.FindReport(params.Get("project"), params.Get("tag"))

	if e != nil {
		log.Printf("missing report (%s, %s): %s", params.Get("project"), params.Get("tag"), e.Error())
		responseWriter.WriteHeader(404)
		fmt.Fprintf(responseWriter, "not-found")
		return
	}

	report, e := parseCoverProfile(txt)

	if e != nil {
		log.Printf("invalid report")
		responseWriter.WriteHeader(404)
		fmt.Fprintf(responseWriter, "not-found")
		return
	}

	color := "414141"

	if report.coverage > 80 {
		color = "green"
	}

	escapedConfig := url.PathEscape(fmt.Sprintf(shieldConfigTemplate, shieldText, report.coverage, color))
	shieldURL, e := url.Parse(fmt.Sprintf(shieldURLTemplate, escapedConfig))

	if e != nil {
		log.Printf("unable to build shield url: %s", e.Error())
		responseWriter.WriteHeader(502)
		return
	}

	shieldQueryParams := url.Values{
		"style": []string{defaultShieldStyle},
	}

	if requestStyle := request.URL.Query().Get("style"); requestStyle != "" {
		shieldQueryParams.Set("style", requestStyle)
	}

	shieldURL.RawQuery = shieldQueryParams.Encode()

	log.Printf("requesting shield: %s", shieldURL)

	client := &http.Client{}
	shieldRequest, e := http.NewRequest("GET", shieldURL.String(), nil)

	if e != nil {
		log.Printf("unable to request shield data: %s", e.Error())
		responseWriter.WriteHeader(502)
		return
	}

	shieldResponse, e := client.Do(shieldRequest)

	if e != nil {
		log.Printf("unable to request shield data: %s", e.Error())
		responseWriter.WriteHeader(502)
		return
	}

	defer shieldResponse.Body.Close()

	cacheValue := fmt.Sprintf("max-age=%d", 10)

	responseWriter.Header().Set("Cache-Control", cacheValue)
	responseWriter.Header().Set("Content-Type", "image/svg+xml")
	responseWriter.WriteHeader(200)
	io.Copy(responseWriter, shieldResponse.Body)
}
