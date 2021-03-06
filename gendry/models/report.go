package models

//go:generate marlowc -input ./report.go

// Report records represent a persisted version of a go coverage report (created from txt and html files)
type Report struct {
	ID         uint    `marlow:"column=id&autoIncrement=true"`
	SystemID   string  `marlow:"column=system_id"`
	ProjectID  string  `marlow:"column=project_id"`
	HTMLFileID string  `marlow:"column=html_file_id"`
	Coverage   float64 `marlow:"column=coverage"`
	Tag        string  `marlow:"column=tag"`
}
