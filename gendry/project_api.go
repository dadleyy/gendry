package gendry

import "log"
import "fmt"
import "net/url"
import "net/http"
import "encoding/json"

// NewProjectAPI creates the api endpoint that is able to create new projects.
func NewProjectAPI(store ProjectStore) APIEndpoint {
	return &projectAPI{store: store}
}

type projectAPI struct {
	notImplementedRoute
	store ProjectStore
}

func (api *projectAPI) Post(writer http.ResponseWriter, request *http.Request, values url.Values) {
	reader := json.NewDecoder(request.Body)
	details := struct {
		Name string `json:"name"`
	}{""}

	if e := reader.Decode(&details); e != nil {
		log.Printf("invalid request to create project: %s", e.Error())
		writer.WriteHeader(422)
		fmt.Fprintf(writer, "invalid request body")
		return
	}

	if details.Name == "" {
		writer.WriteHeader(422)
		fmt.Fprintf(writer, "invalid project name")
		return
	}

	_, key, e := api.store.CreateProject(details.Name)

	if e != nil {
		log.Printf("unable to create project: %s", e.Error())
		writer.WriteHeader(422)
		fmt.Fprintf(writer, "invalid project")
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(201)
	encoder := json.NewEncoder(writer)

	result := struct {
		Name   string `json:"name"`
		APIKey string `json:"api_key"`
	}{details.Name, key}

	if e := encoder.Encode(&result); e != nil {
		log.Printf("unable to write project result: %s", e.Error())
	}
}
