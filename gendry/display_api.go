package gendry

import "io"
import "fmt"
import "log"
import "path"
import "net/url"
import "net/http"

import "github.com/dadleyy/gendry/gendry/models"
import "github.com/dadleyy/gendry/gendry/constants"

// NewDisplayAPI returns a new APIEndpoint capable of returning badge data from shields.io.
func NewDisplayAPI(reports models.ReportStore, projects models.ProjectStore, files FileStore) APIEndpoint {
	api := &displayAPI{
		projects: projects,
		reports:  reports,
		files:    files,
	}
	return api
}

// displayAPI is responsible for writing the svg badge result from shields.io given a report name.
type displayAPI struct {
	notImplementedRoute
	projects models.ProjectStore
	reports  models.ReportStore
	files    FileStore
}

func (a *displayAPI) Get(writer http.ResponseWriter, request *http.Request, params url.Values) {
	matches, e := a.projects.FindProjects(&models.ProjectBlueprint{
		Name: []string{params.Get("project")},
	})

	if e != nil || len(matches) != 1 {
		log.Printf("uanble to find project %s (error: %v) (count: %d)", params.Get("project"), e, len(matches))
		writer.WriteHeader(404)
		fmt.Fprintf(writer, "not-found")
		return
	}

	reports, e := a.reports.FindReports(&models.ReportBlueprint{
		Tag:       []string{params.Get("tag")},
		ProjectID: []string{matches[0].SystemID},
	})

	if e != nil || len(reports) != 1 {
		log.Printf("uanble to find report %s (error: %v) (count: %d)", params.Get("tag"), e, len(reports))
		writer.WriteHeader(404)
		fmt.Fprintf(writer, "not-found")
		return
	}

	if params.Get("format") == "html" {
		a.renderHTML(writer, reports[0])
		return
	}

	color, text := "414141", "generated--coverage"

	if t := request.URL.Query().Get(constants.ShieldTextQueryParam); t != "" {
		text = t
	}

	if reports[0].Coverage > constants.GoodCoverageAmount {
		color = "green"
	}

	escapedConfig := url.PathEscape(fmt.Sprintf(constants.ShieldConfigTemplate, text, reports[0].Coverage, color))
	shieldURL, e := url.Parse(fmt.Sprintf(constants.ShieldURLTemplate, escapedConfig))

	if e != nil {
		log.Printf("unable to build badge url: %s", e.Error())
		writer.WriteHeader(500)
		return
	}

	shieldQueryParams := url.Values{
		"style": []string{constants.DefaultShieldStyle},
	}

	shieldURL.RawQuery = shieldQueryParams.Encode()

	client := &http.Client{}
	shieldRequest, e := http.NewRequest("GET", shieldURL.String(), nil)

	if e != nil {
		log.Printf("unable to load shield: %s", e.Error())
		writer.WriteHeader(404)
		return
	}

	shieldResponse, e := client.Do(shieldRequest)

	if e != nil {
		log.Printf("unable to request shield data: %s", e.Error())
		writer.WriteHeader(502)
		return
	}

	defer shieldResponse.Body.Close()

	cacheValue := fmt.Sprintf("max-age=%d", 10)

	writer.Header().Set("Cache-Control", cacheValue)
	writer.Header().Set("Content-Type", "image/svg+xml")
	writer.WriteHeader(200)
	io.Copy(writer, shieldResponse.Body)
}

func (a *displayAPI) renderHTML(writer http.ResponseWriter, report *models.Report) {
	log.Printf("loading report html for %s", report.SystemID)
	reader, e := a.files.FindFile(path.Join("reports", report.HTMLFileID))

	if e != nil {
		log.Printf("unable to find file for report: %v", e)
		writer.WriteHeader(404)
		return
	}

	writer.Header().Set("Content-Type", "text/html")
	writer.WriteHeader(200)

	defer reader.Close()
	amt, e := io.Copy(writer, reader)

	if e == nil || amt > 0 {
		return
	}

	log.Printf("strange copy on report html, bytes sent: %d (error: %v)", amt, e)
}
