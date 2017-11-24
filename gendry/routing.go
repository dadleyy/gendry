package gendry

import "fmt"
import "log"
import "regexp"
import "strings"
import "net/url"
import "net/http"

// Action types represent a single http request handler, wearere the last url.Values parameter contains path params.
type Action func(http.ResponseWriter, *http.Request, url.Values)

// APIEndpoint represents a single route that is can respond to various HTTP methods.
type APIEndpoint interface {
	Get(http.ResponseWriter, *http.Request, url.Values)
	Post(http.ResponseWriter, *http.Request, url.Values)
	Delete(http.ResponseWriter, *http.Request, url.Values)
}

// RouteList is map of path expressions and their endpoints; matches an incoming request to a single action.
type RouteList map[*regexp.Regexp]APIEndpoint

func (l *RouteList) actionFor(method string, endpoint APIEndpoint) Action {
	switch strings.ToUpper(method) {
	case "POST":
		return endpoint.Post
	case "DELETE":
		return endpoint.Delete
	default:
		return endpoint.Get
	}
}

func (l *RouteList) add(expression *regexp.Regexp, handler APIEndpoint) error {
	if l == nil {
		return fmt.Errorf("invalid-route-list")
	}

	(*l)[expression] = handler
	return nil
}

// Match performs a lookup based on a given http.Reqest record, returning the action associated w/ the path/method.
func (l *RouteList) Match(request *http.Request) (Action, url.Values, bool) {
	path := []byte(request.URL.EscapedPath())

	if l == nil {
		return nil, nil, false
	}

	var fallback Action

	for re, handler := range *l {
		if match := re.Match(path); match != true {
			continue
		}

		if s := re.NumSubexp(); s == 0 {
			fallback = l.actionFor(request.Method, handler)
			continue
		}

		groups := re.FindAllStringSubmatch(string(path), -1)
		names := re.SubexpNames()

		if groups == nil || len(groups) != 1 {
			return l.actionFor(request.Method, handler), make(url.Values), true
		}

		values := groups[0][1:]
		params := make(url.Values)
		count := len(names)

		if count >= 0 {
			names = names[1:]
			count = len(names)
		}

		for indx, v := range values {
			if indx < count && len(names[indx]) >= 1 {
				params.Set(names[indx], v)
				continue
			}

			params.Set(fmt.Sprintf("$%d", indx), v)
		}

		return l.actionFor(request.Method, handler), params, true
	}

	if fallback != nil {
		return fallback, make(url.Values), true
	}

	return nil, nil, false
}

type jsonResponse struct {
	Errors []string `json:"errors"`
}

type notImplementedRoute struct {
}

func (r notImplementedRoute) notImplemented(writer http.ResponseWriter, request *http.Request) {
	log.Printf("not implemented: %s %v", request.Method, request.URL.Path)
	writer.WriteHeader(400)
	fmt.Fprintf(writer, "not-implemented")
}

func (r notImplementedRoute) Delete(writer http.ResponseWriter, request *http.Request, params url.Values) {
	r.notImplemented(writer, request)
}

func (r notImplementedRoute) Post(writer http.ResponseWriter, request *http.Request, params url.Values) {
	r.notImplemented(writer, request)
}

func (r notImplementedRoute) Get(writer http.ResponseWriter, request *http.Request, params url.Values) {
	r.notImplemented(writer, request)
}
