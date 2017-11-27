package gendry

import "io"
import "strings"
import "net/http"

// Runtime defines an interface that is used as the http runtime for the gendry api.
type Runtime interface {
	Start(string, chan<- error)
}

// NewRuntime returns an initialized runtime using the provided route list.
func NewRuntime(routes *RouteList, log LeveledLogger) Runtime {
	r := &runtime{
		LeveledLogger: log,
		routes:        routes,
	}
	return r
}

type runtime struct {
	LeveledLogger
	routes *RouteList
}

func (r *runtime) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	route, params, found := r.routes.Match(request)

	if !found {
		r.Debugf("not found: %v", request.URL)
		responseWriter.WriteHeader(404)
		io.Copy(responseWriter, strings.NewReader("not-found"))
		return
	}

	route(responseWriter, request, params)
}

func (r *runtime) Start(addr string, closed chan<- error) {
	closed <- http.ListenAndServe(addr, r)
}
