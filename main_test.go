package main

import "io"
import "bytes"
import "regexp"
import "testing"
import "strings"
import "net/url"
import "net/http"
import "net/http/httptest"
import "github.com/franela/goblin"

type testRoute struct {
	output io.Reader
	params []url.Values
}

func (r *testRoute) get(writer http.ResponseWriter, request *http.Request, params url.Values) {
	r.respond(writer, request, params)
}

func (r *testRoute) post(writer http.ResponseWriter, request *http.Request, params url.Values) {
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

func Test_parseProfiles(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("atois", func() {
		g.It("returns an error if any are invalid numbers", func() {
			_, e := atois("asdasd")
			g.Assert(e == nil).Equal(false)
		})

		g.It("returns the integer value of the strings if possible", func() {
			r, e := atois("10")
			g.Assert(e).Equal(nil)
			g.Assert(r[0]).Equal(10)
		})
	})

	g.Describe("parseProfiles", func() {
		g.It("returns an error with invalid profile", func() {
			_, e := parseProfiles(strings.NewReader("asdasdasd"))
			g.Assert(e == nil).Equal(false)
		})

		g.It("returns an empty list of profiles if the reader is empty", func() {
			r, e := parseProfiles(strings.NewReader(""))
			g.Assert(e).Equal(nil)
			g.Assert(len(r.files)).Equal(0)
		})

		g.It("returns error if line is poorly formatted", func() {
			_, e := parseProfiles(strings.NewReader(`mode: atomic
			github.com/dadleyy/marlow/examples/library/models/author.marlow.go:11.81,12.52 3 3
			bad-format
			`))
			g.Assert(e == nil).Equal(false)
		})

		g.It("returns error if line has poor values for numerical info", func() {
			_, e := parseProfiles(strings.NewReader(`mode: atomic
			github.com/dadleyy/marlow/examples/library/models/author.marlow.go:11.81,12.52 a basd`))
			g.Assert(e == nil).Equal(false)
		})

		g.It("returns 100%% coverage if that is the case", func() {
			r, e := parseProfiles(strings.NewReader(`mode: atomic
			github.com/dadleyy/marlow/examples/library/models/author.marlow.go:11.81,12.52 3 3`))
			g.Assert(e).Equal(nil)
			g.Assert(len(r.files)).Equal(1)
			g.Assert(r.coverage).Equal(100)
		})
	})

	g.Describe("server", func() {
		scaffold := struct {
			server *server
			writer *httptest.ResponseRecorder
		}{}

		g.BeforeEach(func() {
			scaffold.server = &server{}
			scaffold.writer = httptest.NewRecorder()
		})

		g.It("writes a 404 response if no route matched", func() {
			request := httptest.NewRequest("GET", "/bad-route", new(bytes.Buffer))
			scaffold.server.ServeHTTP(scaffold.writer, request)
			g.Assert(scaffold.writer.Code).Equal(404)
		})

		g.Describe("with a valid route list", func() {
			var r *testRoute

			g.BeforeEach(func() {
				r = &testRoute{}
				scaffold.server.routes = &routeList{}
			})

			g.It("delegates to the matched route if match found", func() {
				scaffold.server.routes.add(regexp.MustCompile("^/bad-route"), r)
				request := httptest.NewRequest("GET", "/bad-route", new(bytes.Buffer))
				scaffold.server.ServeHTTP(scaffold.writer, request)
				g.Assert(scaffold.writer.Code).Equal(200)
			})

			g.It("delegates to the matched route if match found (including url values)", func() {
				scaffold.server.routes.add(regexp.MustCompile("^/bad-route/(.*)"), r)
				request := httptest.NewRequest("GET", "/bad-route/123", new(bytes.Buffer))
				scaffold.server.ServeHTTP(scaffold.writer, request)
				g.Assert(scaffold.writer.Code).Equal(200)
				g.Assert(r.params[0].Get("$0")).Equal("123")
			})

			g.It("delegates to the matched route if match found (including named values)", func() {
				scaffold.server.routes.add(regexp.MustCompile("^/bad-route/(?P<uuid>.*)"), r)
				request := httptest.NewRequest("GET", "/bad-route/123", new(bytes.Buffer))
				scaffold.server.ServeHTTP(scaffold.writer, request)
				g.Assert(scaffold.writer.Code).Equal(200)
				g.Assert(r.params[0].Get("uuid")).Equal("123")
			})
		})
	})
}
