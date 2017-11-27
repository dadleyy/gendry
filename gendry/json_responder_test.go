package gendry

import "testing"
import "encoding/json"
import "net/http/httptest"
import "github.com/franela/goblin"

func Test_jsonResponder(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("jsonResponder test suite", func() {
		var r jsonResponder
		var o *httptest.ResponseRecorder

		g.BeforeEach(func() {
			o = httptest.NewRecorder()
		})

		g.It("renders out a successful json response with meta, errors and data keys", func() {
			r.renderSuccess(o, struct {
				Name string `json:"name"`
			}{"danny"})
			decoder := json.NewDecoder(o.Body)
			expected := struct {
				Metadata map[string]interface{} `json:"meta"`
				Errors   []string               `json:"errors"`
				Data     []struct {
					Name string `json:"name"`
				} `json:"data"`
			}{}
			e := decoder.Decode(&expected)

			g.Assert(e).Equal(nil)
			g.Assert(expected.Data[0].Name).Equal("danny")
			g.Assert(len(expected.Errors)).Equal(0)
			g.Assert(expected.Metadata["limit"]).Equal(nil)
		})

		g.It("renders out a successful json response with meta, errors and data keys (including paging)", func() {
			u := struct {
				Name string `json:"name"`
			}{"danny"}

			r.renderSuccess(o, u, pagingInfo{10, 10, 10})

			decoder := json.NewDecoder(o.Body)

			expected := struct {
				Metadata map[string]interface{} `json:"meta"`
				Errors   []string               `json:"errors"`
				Data     []struct {
					Name string `json:"name"`
				} `json:"data"`
			}{}
			e := decoder.Decode(&expected)

			g.Assert(e).Equal(nil)
			g.Assert(expected.Metadata["limit"]).Equal(10)
			g.Assert(expected.Metadata["offset"]).Equal(10)
			g.Assert(expected.Metadata["total"]).Equal(10)
			g.Assert(expected.Data[0].Name).Equal("danny")
			g.Assert(len(expected.Errors)).Equal(0)
		})

		g.It("renders out the error json response structure", func() {
			r.renderError(o, "bad-request")
			decoder := json.NewDecoder(o.Body)
			expected := struct {
				Errors []string `json:"errors"`
			}{}
			e := decoder.Decode(&expected)
			g.Assert(e).Equal(nil)
			g.Assert(expected.Errors[0]).Equal("bad-request")
		})
	})
}
