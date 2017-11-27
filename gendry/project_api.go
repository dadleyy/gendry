package gendry

import "io"
import "fmt"
import "bytes"
import "strconv"
import "net/url"
import "net/http"
import "encoding/json"
import "github.com/satori/go.uuid"
import "github.com/dadleyy/gendry/gendry/models"
import "github.com/dadleyy/gendry/gendry/constants"

// NewProjectAPI creates the api endpoint that is able to create new projects.
func NewProjectAPI(store models.ProjectStore, log LeveledLogger) APIEndpoint {
	api := &projectAPI{
		LeveledLogger: log,
		store:         store,
	}

	return api
}

type projectAPI struct {
	LeveledLogger
	jsonResponder
	store models.ProjectStore
}

func (a *projectAPI) Get(writer http.ResponseWriter, request *http.Request, params url.Values) {
	paging := a.paging(request)

	blueprint := &models.ProjectBlueprint{
		Offset: paging.offset,
		Limit:  paging.limit,
	}

	projects, e := a.store.FindProjects(blueprint)

	if e != nil {
		a.Warnf("unable to find projects (error %v)", e)
		a.renderError(writer, "server-error")
		return
	}

	results := make([]interface{}, len(projects))

	for i, p := range projects {
		item := struct {
			ID       uint   `json:"id"`
			SystemID string `json:"system_id"`
			Name     string `json:"name"`
		}{p.ID, p.SystemID, p.Name}

		results[i] = item
	}

	paging.total, e = a.store.CountProjects(blueprint)

	if e != nil {
		a.Warnf("unable to find projects (error %v)", e)
		a.renderError(writer, "server-error")
		return
	}

	a.renderSuccess(writer, append(results, paging)...)
}

func (a *projectAPI) Delete(writer http.ResponseWriter, request *http.Request, params url.Values) {
	projectID := request.URL.Query().Get(constants.ProjectIDParamName)

	projects, e := a.store.FindProjects(&models.ProjectBlueprint{
		Token: []string{request.Header.Get(constants.ProjectAuthTokenAPIHeader)},
	})

	if e != nil || len(projects) != 1 {
		a.Warnf("invalid project %s (error %v)", request.Header.Get(constants.ProjectAuthTokenAPIHeader), e)
		a.renderError(writer, "invalid-project")
		return
	}

	if projects[0].SystemID != projectID && fmt.Sprintf("%d", projects[0].ID) != projectID {
		a.Warnf("invalid project %s (found %v)", projectID, projects[0].SystemID)
		a.renderError(writer, "invalid-project")
		return
	}

	blueprint := &models.ProjectBlueprint{
		SystemID: []string{projects[0].SystemID},
	}

	if _, e := a.store.DeleteProjects(blueprint); e != nil {
		a.Errorf("unable to delete project %s (error %v)", projects[0].SystemID, e)
		a.renderError(writer, "server-error")
		return
	}

	a.Infof("deleted project %s (id %s)", projects[0].Name, projects[0].SystemID)

	a.renderSuccess(writer, nil)
	return
}

func (a *projectAPI) Post(writer http.ResponseWriter, request *http.Request, params url.Values) {
	defer request.Body.Close()
	decoder := json.NewDecoder(request.Body)
	project := struct {
		Name string `json:"name"`
	}{}

	if e := decoder.Decode(&project); e != nil {
		a.renderError(writer, "invalid-project")
		return
	}

	c, e := a.store.CountProjects(&models.ProjectBlueprint{
		Name: []string{project.Name},
	})

	if e != nil {
		a.Warnf("invalid count: %s", e.Error())
		a.renderError(writer, "invalid-project")
		return
	}

	if c != 0 {
		a.Warnf("duplicate project: %s", project.Name)
		a.renderError(writer, "invalid-project")
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
		a.Warnf("unable to create project: %s", e.Error())
		a.renderError(writer, "invalid-project")
		return
	}

	a.Infof("created new project %s (id %s)", project.Name, systemID)

	a.renderSuccess(writer, struct {
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

func (a *projectAPI) paging(request *http.Request) pagingInfo {
	paging := pagingInfo{limit: 10, offset: 0}

	if offset, e := strconv.Atoi(request.URL.Query().Get(constants.OffsetParamName)); e == nil {
		paging.offset = offset
	}

	if limit, e := strconv.Atoi(request.URL.Query().Get(constants.LimitParamName)); e == nil {
		paging.limit = limit
	}

	return paging
}
