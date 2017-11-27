package gendry

import "time"
import "net/http"
import "encoding/json"

type jsonResponder struct {
}

func (r jsonResponder) renderSuccess(writer http.ResponseWriter, data ...interface{}) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(200)
	meta := make(map[string]interface{})
	meta["time"] = time.Now()

	results := make([]interface{}, 0, len(data))

	for _, item := range data {
		paging, ok := item.(pagingInfo)

		if ok {
			meta["total"] = paging.total
			meta["offset"] = paging.offset
			meta["limit"] = paging.limit
			continue
		}

		results = append(results, item)
	}

	response := struct {
		Metadata map[string]interface{} `json:"meta"`
		Errors   []string               `json:"errors"`
		Results  []interface{}          `json:"data"`
	}{meta, nil, results}

	encoder := json.NewEncoder(writer)
	encoder.Encode(&response)
}

func (r jsonResponder) renderError(writer http.ResponseWriter, errors ...string) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(422)
	meta := make(map[string]interface{})

	meta["time"] = time.Now()

	response := struct {
		Metadata map[string]interface{} `json:"meta"`
		Errors   []string               `json:"errors"`
		Results  []interface{}          `json:"data"`
	}{meta, errors, nil}

	encoder := json.NewEncoder(writer)
	encoder.Encode(&response)
}
