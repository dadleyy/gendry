package gendry

import "github.com/dadleyy/gendry/gendry/models"

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
func NewReportAPI(reports models.ReportStore, projects models.ProjectStore) APIEndpoint {
	return &reportAPI{reports: reports, projects: projects}
}

type reportAPI struct {
	notImplementedRoute
	projects models.ProjectStore
	reports  models.ReportStore
}
