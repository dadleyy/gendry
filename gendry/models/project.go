package models

//go:generate marlowc -input ./project.go

// Project records represent a set of reports - each with an auth token for management.
type Project struct {
	ID       uint   `marlow:"column=id&autoIncrement=true"`
	Name     string `marlow:"column=name"`
	SystemID string `marlow:"column=system_id"`
	Token    string `marlow:"column=auth_token"`
}
