package constants

const (
	// ProjectAuthTokenAPIHeader is the header name used to authenticate project requests.
	ProjectAuthTokenAPIHeader = "x-project-auth"

	// DisplayAPIRegex is the regular expression used to match requests to the display api
	DisplayAPIRegex = "^/reports/(?P<project>[\\w\\/]+)/(?P<tag>[A-z0-9]+)\\.(?P<format>html|svg)"

	// ProjectIDParamName is used as the key wherever a project id is expected.
	ProjectIDParamName = "project_id"

	// ReportIDParamName is used as the key wherever a report id is expected.
	ReportIDParamName = "report_id"

	// ReportProjectIDBodyParam is the body param key that will be used as the project id in the report upload request.
	ReportProjectIDBodyParam = "project_id"

	// ReportFileBodyParam is the body param key that will be used to load files for a given report.
	ReportFileBodyParam = "files"

	// ShieldTextQueryParam is used as a query param key that, if provided, will determine which text to display.
	ShieldTextQueryParam = "text"
)
