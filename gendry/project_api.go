package gendry

import "io"
import "log"
import "fmt"
import "bytes"
import "net/url"
import "net/http"
import "encoding/json"
import "github.com/satori/go.uuid"
import "github.com/dadleyy/gendry/gendry/models"
import "github.com/dadleyy/gendry/gendry/constants"

// NewProjectAPI creates the api endpoint that is able to create new projects.
func NewProjectAPI(store models.ProjectStore) APIEndpoint {
	return &projectAPI{store: store}
}

type projectAPI struct {
	notImplementedRoute
	jsonResponder
	store models.ProjectStore
}

func (a *projectAPI) Delete(writer http.ResponseWriter, request *http.Request, params url.Values) {
	projectID := request.URL.Query().Get(constants.ProjectIDParamName)

	projects, e := a.store.FindProjects(&models.ProjectBlueprint{
		Token: []string{request.Header.Get(constants.ProjectAuthTokenAPIHeader)},
	})

	if e != nil || len(projects) != 1 {
		log.Printf("invalid project %s (error %v)", request.Header.Get(constants.ProjectAuthTokenAPIHeader), e)
		a.error(writer, "invalid-project")
		return
	}

	if projects[0].SystemID != projectID && fmt.Sprintf("%d", projects[0].ID) != projectID {
		log.Printf("invalid project %s (found %v)", projectID, projects[0].SystemID)
		a.error(writer, "invalid-project")
		return
	}

	blueprint := &models.ProjectBlueprint{
		SystemID: []string{projects[0].SystemID},
	}

	if _, e := a.store.DeleteProjects(blueprint); e != nil {
		log.Printf("unable to delete project %s (error %v)", projects[0].SystemID, e)
		a.error(writer, "server-error")
		return
	}

	a.success(writer, nil)
	return
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
