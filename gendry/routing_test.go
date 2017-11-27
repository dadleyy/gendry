package gendry

import "io"
import "bytes"
import "regexp"
import "net/url"
import "testing"
import "net/http"
import "net/http/httptest"
import "github.com/franela/goblin"

type testRoute struct {
	output io.Reader
	params []url.Values
}

func (r *testRoute) Delete(writer http.ResponseWriter, request *http.Request, params url.Values) {
}

func (r *testRoute) Get(writer http.ResponseWriter, request *http.Request, params url.Values) {
	r.respond(writer, request, params)
}

func (r *testRoute) Post(writer http.ResponseWriter, request *http.Request, params url.Values) {
	r.respond(writer, request, params)
}

func (r *testRoute) respond(writer http.ResponseWriter, request *http.Request, params url.Values) {
	writer.WriteHeader(200)

	if r.params == nil {
		r.params = make([]url.Values, 0, 1)
	}

	r.params = append(r.params, params)

	if r.output != nil {
		io.Copy(writer, r.output)
	}
}

func Test_RouteList(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("RouteList", func() {
		var routes *RouteList
		var route *testRoute

		g.BeforeEach(func() {
			route = &testRoute{}
			routes = &RouteList{}
		})

		g.It("returns false if no route found", func() {
			request := httptest.NewRequest("GET", "/bad-route", new(bytes.Buffer))
			_, _, found := routes.Match(request)
			g.Assert(found).Equal(false)
		})

		g.It("returns true if route found, with an empty path param map", func() {
			routes.add(regexp.MustCompile("^/bad-route"), route)
			request := httptest.NewRequest("GET", "/bad-route", new(bytes.Buffer))
			_, values, found := routes.Match(request)
			g.Assert(found).Equal(true)
			g.Assert(len(values)).Equal(0)
		})

		g.It("returns true if route found, with unnamed path param matches in the value list", func() {
			routes.add(regexp.MustCompile("^/bad-route/(.*)"), route)
			request := httptest.NewRequest("GET", "/bad-route/213", new(bytes.Buffer))
			_, values, found := routes.Match(request)
			g.Assert(found).Equal(true)
			g.Assert(len(values)).Equal(1)
			g.Assert(values.Get("$0")).Equal("213")
		})

		g.It("returns true if route found, with named path param matches in the value list", func() {
			routes.add(regexp.MustCompile("^/bad-route/(?P<uuid>.*)"), route)
			request := httptest.NewRequest("GET", "/bad-route/213", new(bytes.Buffer))
			_, values, found := routes.Match(request)
			g.Assert(found).Equal(true)
			g.Assert(len(values)).Equal(1)
			g.Assert(values.Get("uuid")).Equal("213")
		})
	})
}
