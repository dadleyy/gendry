package models

//go:generate marlowc -input ./file.go

type File struct {
	ID       uint   `marlow:"column=id&autoIncrement=true"`
	SystemID string `marlow:"column=system_id"`
	Status   string `marlow:"column=status"`
}
