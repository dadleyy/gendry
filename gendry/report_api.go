package gendry

import "io"
import "fmt"
import "path"
import "strconv"
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
func NewReportAPI(re models.ReportStore, pr models.ProjectStore, fs FileStore, log LeveledLogger) APIEndpoint {
	api := &reportAPI{
		LeveledLogger: log,
		filestore:     fs,
		reports:       re,
		projects:      pr,
	}

	return api
}

type reportAPI struct {
	LeveledLogger
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

func (a *reportAPI) Get(writer http.ResponseWriter, request *http.Request, params url.Values) {
	project, e := a.project(request)

	if e != nil {
		a.Warnf("unable to find project (error %s)", e.Error())
		a.renderError(writer, "invalid-project")
		return
	}

	target := request.URL.Query().Get(constants.ProjectIDParamName)

	paging := a.paging(request)

	blueprint := &models.ProjectBlueprint{
		SystemID: []string{target},
	}

	if internal, e := strconv.Atoi(target); e == nil {
		blueprint.ID = []uint{uint(internal)}
		blueprint.Inclusive = true
	}

	matches, e := a.projects.FindProjects(blueprint)

	if e != nil || len(matches) != 1 {
		a.renderError(writer, "not-found")
		return
	}

	if matches[0].SystemID != project.SystemID {
		a.renderError(writer, "invalid-project")
		return
	}

	bp := &models.ReportBlueprint{
		ProjectID: []string{matches[0].SystemID},
		Limit:     paging.limit,
		Offset:    paging.offset,
	}

	reports, e := a.reports.FindReports(bp)

	if e != nil {
		a.Warnf("unable to find reports for project %s (error %v)", matches[0].SystemID, e)
		a.renderError(writer, "server-error")
		return
	}

	paging.total, e = a.reports.CountReports(bp)

	if e != nil {
		a.Warnf("unable to find reports for project %s (error %v)", matches[0].SystemID, e)
		a.renderError(writer, "server-error")
		return
	}

	results := make([]interface{}, len(reports))

	for i, r := range reports {
		results[i] = struct {
			ID         uint    `json:"id"`
			SystemID   string  `json:"system_id"`
			HTMLFileID string  `json:"html_field_id"`
			ProjectID  string  `json:"project_id"`
			Tag        string  `json:"tag"`
			Coverage   float64 `json:"coverage"`
		}{r.ID, r.SystemID, r.HTMLFileID, r.ProjectID, r.Tag, r.Coverage}
	}

	a.renderSuccess(writer, append(results, paging)...)
}

func (a *reportAPI) Delete(writer http.ResponseWriter, request *http.Request, params url.Values) {
	report, e := a.authorizeLookup(request)

	if e != nil {
		a.Warnf("unauthorized attempt (error %v)", e)
		a.renderError(writer, "invalid-report")
		return
	}

	blueprint := &models.ReportBlueprint{
		SystemID: []string{report.SystemID},
	}

	if _, e := a.reports.DeleteReports(blueprint); e != nil {
		a.Warnf("unable to delete report (error %v)", e)
		a.renderError(writer, "server-error")
		return
	}

	a.renderSuccess(writer, nil)
}

func (a *reportAPI) Post(writer http.ResponseWriter, request *http.Request, params url.Values) {
	project, e := a.project(request)

	if e != nil {
		a.Warnf("unable to find project (error %v) (header %v)", e, request.Header)
		a.renderError(writer, "not-found")
		return
	}

	if e := request.ParseMultipartForm(maxReportFileSize); e != nil {
		a.renderError(writer, "invalid-request")
		return
	}

	projectID, tag := request.Form.Get("project_id"), request.Form.Get("tag")

	if fmt.Sprintf("%d", project.ID) != projectID && project.SystemID != projectID {
		a.Warnf("requested project != authed (request: %s, auth: %d)", projectID, project.ID)
		a.renderError(writer, "invalid-project")
		return
	}

	if tag == "" {
		a.renderError(writer, "invalid-tag")
		return
	}

	reports, e := a.parseReportForm(request.MultipartForm)

	if e != nil {
		a.Warnf("unable to parse request body for creating report in project %s (error %v)", projectID, e)
		a.renderError(writer, e.Error())
		return
	}

	fileID, e := a.writeReportHTMLFile(reports.html)

	if e != nil {
		a.Warnf("unable to allocate new file: %s (id: %s)", e.Error(), fileID)
		a.renderError(writer, "server-error")
		return
	}

	record := models.Report{
		SystemID:   fmt.Sprintf("%s", uuid.NewV4()),
		HTMLFileID: fileID,
		Coverage:   reports.coverage.coverage,
		ProjectID:  project.SystemID,
		Tag:        tag,
	}

	if _, e := a.reports.CreateReports(record); e != nil {
		a.Errorf("unable to save report: %s", e.Error())
		a.renderError(writer, e.Error())
		return
	}

	primaryIDs, e := a.reports.SelectIDs(&models.ReportBlueprint{SystemID: []string{record.SystemID}})

	if e != nil {
		a.renderError(writer, "invalid-report")
		return
	}

	a.Infof("successfully created report (id %s) - coverage %f", record.SystemID, record.Coverage)

	a.renderSuccess(writer, struct {
		ID         uint    `json:"id"`
		SystemID   string  `json:"system_id"`
		Tag        string  `json:"tag"`
		HTMLFileID string  `json:"html_file_id"`
		Coverage   float64 `json:"coverage"`
		ProjectID  string  `json:"project_id"`
	}{primaryIDs[0], record.SystemID, record.Tag, record.HTMLFileID, record.Coverage, record.ProjectID})
}

func (a *reportAPI) project(request *http.Request) (*models.Project, error) {
	token := request.Header.Get(constants.ProjectAuthTokenAPIHeader)
	projects, e := a.projects.FindProjects(&models.ProjectBlueprint{Token: []string{token}})

	if e != nil {
		return nil, e
	}

	if len(projects) != 1 {
		return nil, fmt.Errorf("invalid-token")
	}

	return projects[0], nil
}

func (a *reportAPI) authorizeLookup(request *http.Request) (*models.Report, error) {
	project, e := a.project(request)

	if e != nil {
		return nil, e
	}

	id := request.URL.Query().Get(constants.ReportIDParamName)

	blueprint := &models.ReportBlueprint{
		SystemID: []string{id},
	}

	if internal, e := strconv.Atoi(id); e == nil {
		blueprint.ID = []uint{uint(internal)}
		blueprint.Inclusive = true
	}

	reports, e := a.reports.FindReports(blueprint)

	if len(reports) != 1 || e != nil {
		return nil, fmt.Errorf("invalid-report")
	}

	if reports[0].ProjectID != project.SystemID {
		return nil, fmt.Errorf("unauthorized")
	}

	return reports[0], nil
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
			a.Warnf("received strange filetype during report creation: %s", ext)
			continue
		}

		if ext == ".html" {
			result.html = f
			continue
		}

		coverage, e := f.Open()

		if e != nil {
			a.Warnf("unable to open coverage file during report creation: %s", e.Error())
			return nil, fmt.Errorf("invalid-coverage")
		}

		defer coverage.Close()
		result.coverage, e = parseCoverProfile(coverage)

		if e != nil {
			a.Warnf("unable to open coverage file during report creation: %s", e.Error())
			return nil, fmt.Errorf("invalid-coverage")
		}
	}

	if result.coverage == nil || result.html == nil {
		return nil, fmt.Errorf("invalid-files %d", len(files))
	}

	return result, nil
}

func (a *reportAPI) paging(request *http.Request) pagingInfo {
	paging := pagingInfo{limit: 10, offset: 0}

	if offset, e := strconv.Atoi(request.URL.Query().Get(constants.OffsetParamName)); e == nil {
		paging.offset = offset
	}

	if limit, e := strconv.Atoi(request.URL.Query().Get(constants.LimitParamName)); e == nil {
		paging.limit = limit
	}

	return paging
}
