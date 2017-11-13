package gendry

import "log"
import "fmt"
import "net/url"
import "net/http"
import "encoding/json"
import "github.com/satori/go.uuid"
import "github.com/dadleyy/gendry/gendry/models"

// NewProjectAPI creates the api endpoint that is able to create new projects.
func NewProjectAPI(store models.ProjectStore) APIEndpoint {
	return &projectAPI{store: store}
}

type projectAPI struct {
	notImplementedRoute
	store models.ProjectStore
}

func (a *projectAPI) Post(writer http.ResponseWriter, request *http.Request, params url.Values) {
	defer request.Body.Close()
	decoder := json.NewDecoder(request.Body)
	project := struct {
		Name string `json:"name"`
	}{}

	if e := decoder.Decode(&project); e != nil {
		writer.WriteHeader(422)
		fmt.Fprintf(writer, "invalid project")
		return
	}

	c, e := a.store.CountProjects(&models.ProjectBlueprint{
		Name: []string{project.Name},
	})

	if e != nil {
		log.Printf("invalid count: %s", e.Error())
		writer.WriteHeader(500)
		fmt.Fprintf(writer, "server error")
		return
	}

	if c != 0 {
		log.Printf("duplicate project: %s", project.Name)
		writer.WriteHeader(422)
		fmt.Fprintf(writer, "invalid project")
		return
	}

	systemID := uuid.NewV4().String()

	id, e := a.store.CreateProjects(models.Project{
		Name:     project.Name,
		SystemID: systemID,
	})

	if e != nil {
		log.Printf("unable to create project: %s", e.Error())
		writer.WriteHeader(422)
		fmt.Fprintf(writer, "invalid project")
		return
	}

	writer.Header().Set("Content-Type", "application/json")

	output := json.NewEncoder(writer)

	output.Encode(&struct {
		ID       int64  `json:"id"`
		Name     string `json:"name"`
		SystemID string `json:"system_id"`
	}{id, project.Name, systemID})
}
