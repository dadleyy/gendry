package gendry

import "io"
import "fmt"
import "log"
import "path"
import "bytes"
import "strings"
import "net/url"
import "net/http"
import "encoding/json"
import "mime/multipart"

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
func NewReportAPI(store ProjectStore) APIEndpoint {
	return &reportAPI{store: store}
}

type reportAPI struct {
	notImplementedRoute
	store ProjectStore
}

func (api *reportAPI) Post(writer http.ResponseWriter, request *http.Request, values url.Values) {
	if e := request.ParseMultipartForm(maxReportFileSize); e != nil {
		writer.WriteHeader(422)
		fmt.Fprintf(writer, "unable to parse request: %s", e.Error())
		return
	}

	log.Printf("received report upload request")

	defer request.Body.Close()

	if form := request.MultipartForm; form == nil || form.File[reportFileBodyParam] == nil {
		writer.WriteHeader(422)
		fmt.Fprintf(writer, "missing report file")
		return
	}

	project, e := api.loadProject(request)

	if e != nil {
		writer.WriteHeader(422)
		fmt.Fprintf(writer, "invalid project: %s", e.Error())
		return
	}

	tag := request.Form.Get(reportTagBodyParam)

	if tag == "" {
		writer.WriteHeader(422)
		fmt.Fprintf(writer, "missing tag name")
		return
	}

	report, e := api.parseFormBody(request.MultipartForm)

	if e != nil {
		writer.WriteHeader(422)
		fmt.Fprintf(writer, "invalid report request: %s", e.Error())
		return
	}

	if e := project.StoreReport(tag, report[htmlCoverageFileExtension], report[textCoverageFileExtension]); e != nil {
		writer.WriteHeader(500)
		fmt.Fprintf(writer, "unable to store report: %s", e.Error())
		return
	}

	jsonWriter := json.NewEncoder(writer)

	response := struct {
		Tag string `json:"tag"`
	}{tag}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(202)

	if e := jsonWriter.Encode(&response); e != nil {
		log.Printf("error writing json: %s", e.Error())
	}
}

func (api *reportAPI) parseFormBody(form *multipart.Form) (map[string]string, error) {
	files := form.File
	report := make(map[string]string)

	for _, header := range files[reportFileBodyParam] {
		fileType := path.Ext(header.Filename)

		if fileType != textCoverageFileExtension && fileType != htmlCoverageFileExtension {
			continue
		}

		file, e := header.Open()

		if e != nil {
			log.Printf("invalid report file: %s", e.Error())
			continue
		}

		defer file.Close()

		if _, dupe := report[fileType]; dupe == true {
			return nil, fmt.Errorf("invalid report request value: may only contain a single %s file", fileType)
		}

		buffer := new(bytes.Buffer)

		if _, e := io.Copy(buffer, file); e != nil {
			return nil, fmt.Errorf("unable to copy contents of file into memory: %s", e.Error())
		}

		if fileType == ".html" {
			report[fileType] = buffer.String()
			continue
		}

		if _, e := parseCoverProfile(strings.NewReader(buffer.String())); e != nil {
			return nil, fmt.Errorf("invalid report request value: invalid coverage.txt file: %s", e.Error())
		}

		log.Printf("done parsing report text file: %s", header.Filename)
		report[fileType] = buffer.String()
	}

	if len(report) != 2 {
		return nil, fmt.Errorf("invalid report request value: must contain html and txt file")
	}

	return report, nil
}

func (api *reportAPI) loadProject(request *http.Request) (Project, error) {
	token := request.Header.Get(projectAPIKeyHeader)
	projectIdentifier := request.Form.Get(reportProjectBodyParam)

	if projectIdentifier == "" {
		return nil, fmt.Errorf("invalid-project")
	}

	if token == "" {
		return nil, fmt.Errorf("invalid-token")
	}

	return api.store.FindProject(projectIdentifier, token)
}
