package models

//go:generate marlowc -input ./report.go

type Report struct {
	ID                 uint    `marlow:"column=id&autoIncrement=true"`
	Name               string  `marlow:"column=name"`
	SystemID           string  `marlow:"column=system_id"`
	ProjectID          uint    `marlow:"column=project_id"`
	CoveragePercentage float64 `marlow:"column=coverage_percentage"`
}
