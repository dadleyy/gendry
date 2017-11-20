package gendry

import "io"
import "log"
import "fmt"
import "path"
import "net/url"
import "net/http"
import "mime/multipart"
import "github.com/satori/go.uuid"
import "github.com/dadleyy/gendry/gendry/models"
import "github.com/dadleyy/gendry/gendry/constants"

const (
	maxReportFileSize         = 2048
	reportFileBodyParam       = "files"
	reportProjectBodyParam    = "project"
	reportTagBodyParam        = "tag"
	textCoverageFileExtension = ".txt"
	htmlCoverageFileExtension = ".html"
	projectAPIKeyHeader       = "x-project-key"
)

// NewReportAPI returns an api for storing and retreiving reports
func NewReportAPI(reports models.ReportStore, projects models.ProjectStore, files FileStore) APIEndpoint {
	return &reportAPI{
		filestore: files,
		reports:   reports,
		projects:  projects,
	}
}

type reportAPI struct {
	notImplementedRoute
	jsonResponder
	filestore FileStore
	projects  models.ProjectStore
	reports   models.ReportStore
}

type reportFiles struct {
	html     *multipart.FileHeader
	coverage *reportProfile
}

func (a *reportAPI) Post(writer http.ResponseWriter, request *http.Request, params url.Values) {
	projectToken := request.Header.Get(constants.ProjectAuthTokenAPIHeader)

	matchingProjects, e := a.projects.FindProjects(&models.ProjectBlueprint{
		Token: []string{projectToken},
	})

	if e != nil || len(matchingProjects) == 0 {
		log.Printf("invalid report request, token[%s], err[%v]", projectToken, e)
		a.error(writer, "invalid-project")
		return
	}

	if e := request.ParseMultipartForm(maxReportFileSize); e != nil {
		a.error(writer, "invalid-request")
		return
	}

	projectID, tag := request.Form.Get("project_id"), request.Form.Get("tag")

	if fmt.Sprintf("%d", matchingProjects[0].ID) != projectID && matchingProjects[0].SystemID != projectID {
		log.Printf("requested project != authed (request: %s, auth: %d)", projectID, matchingProjects[0].ID)
		a.error(writer, "invalid-project")
		return
	}

	if tag == "" {
		a.error(writer, "invalid-tag")
		return
	}

	reports, e := a.parseReportForm(request.MultipartForm)

	if e != nil {
		a.error(writer, e.Error())
		return
	}

	log.Printf("valid report files, coverage: %v", reports.coverage.coverage)

	fileID, e := a.writeReportHTMLFile(reports.html)

	if e != nil {
		log.Printf("unable to allocate new file: %s (id: %s)", e.Error(), fileID)
		a.error(writer, "server-error")
		return
	}

	record := models.Report{
		SystemID:   fmt.Sprintf("%s", uuid.NewV4()),
		HTMLFileID: fileID,
		Coverage:   reports.coverage.coverage,
		ProjectID:  matchingProjects[0].SystemID,
		Tag:        tag,
	}

	if _, e := a.reports.CreateReports(record); e != nil {
		log.Printf("unable to save report: %s", e.Error())
		a.error(writer, e.Error())
		return
	}

	primaryIDs, e := a.reports.SelectIDs(&models.ReportBlueprint{SystemID: []string{record.SystemID}})

	if e != nil {
		a.error(writer, "invalid-report")
		return
	}

	a.success(writer, struct {
		ID         uint    `json:"id"`
		SystemID   string  `json:"system_id"`
		Tag        string  `json:"tag"`
		HTMLFileID string  `json:"html_file_id"`
		Coverage   float64 `json:"coverage"`
		ProjectID  string  `json:"project_id"`
	}{primaryIDs[0], record.SystemID, record.Tag, record.HTMLFileID, record.Coverage, record.ProjectID})
}

func (a *reportAPI) writeReportHTMLFile(source *multipart.FileHeader) (string, error) {
	id, file, e := a.filestore.NewFile("text/html", "reports")

	if e != nil {
		return "", e
	}

	defer file.Close()

	reader, e := source.Open()

	if e != nil {
		return "", e
	}

	defer reader.Close()

	size, e := io.Copy(file, reader)

	if e != nil {
		return "", e
	}

	if size > 0 != true {
		return "", fmt.Errorf("no-upload")
	}

	return id, nil
}

func (a *reportAPI) parseReportForm(form *multipart.Form) (*reportFiles, error) {
	files := form.File[constants.ReportFileBodyParam]
	result := &reportFiles{}

	for _, f := range files {
		ext := path.Ext(f.Filename)

		if ext != ".html" && ext != ".txt" {
			log.Printf("received strange filetype during report creation: %s", ext)
			continue
		}

		if ext == ".html" {
			result.html = f
			continue
		}

		coverage, e := f.Open()

		if e != nil {
			log.Printf("unable to open coverage file during report creation: %s", e.Error())
			return nil, fmt.Errorf("invalid-coverage")
		}

		defer coverage.Close()
		result.coverage, e = parseCoverProfile(coverage)

		if e != nil {
			log.Printf("unable to open coverage file during report creation: %s", e.Error())
			return nil, fmt.Errorf("invalid-coverage")
		}
	}

	if result.coverage == nil || result.html == nil {
		return nil, fmt.Errorf("invalid-files")
	}

	return result, nil
}
