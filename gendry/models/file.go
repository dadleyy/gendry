package models

//go:generate marlowc -input ./file.go

// File records provide a database lookup for persisted files on the filestore.
type File struct {
	ID       uint   `marlow:"column=id&autoIncrement=true"`
	SystemID string `marlow:"column=system_id"`
	Status   string `marlow:"column=status"`
}
