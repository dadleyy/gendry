package gendry

import "io"
import "log"
import "bytes"
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
	jsonResponder
	store models.ProjectStore
}

func (a *projectAPI) Post(writer http.ResponseWriter, request *http.Request, params url.Values) {
	defer request.Body.Close()
	decoder := json.NewDecoder(request.Body)
	project := struct {
		Name string `json:"name"`
	}{}

	if e := decoder.Decode(&project); e != nil {
		a.error(writer, "invalid-project")
		return
	}

	c, e := a.store.CountProjects(&models.ProjectBlueprint{
		Name: []string{project.Name},
	})

	if e != nil {
		log.Printf("invalid count: %s", e.Error())
		a.error(writer, "invalid-project")
		return
	}

	if c != 0 {
		log.Printf("duplicate project: %s", project.Name)
		a.error(writer, "invalid-project")
		return
	}

	systemID := uuid.NewV4().String()
	token := a.generateToken()

	id, e := a.store.CreateProjects(models.Project{
		Name:     project.Name,
		SystemID: systemID,
		Token:    token,
	})

	if e != nil {
		log.Printf("unable to create project: %s", e.Error())
		a.error(writer, "invalid-project")
		return
	}

	a.success(writer, struct {
		ID       int64  `json:"id"`
		SystemID string `json:"system_id"`
		Token    string `json:"token"`
		Name     string `json:"name"`
	}{id, systemID, token, project.Name})
}

func (a *projectAPI) generateToken() string {
	output := new(bytes.Buffer)
	io.Copy(output, newTokenGenerator(20))
	return output.String()
}
