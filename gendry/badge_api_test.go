package gendry

import "testing"
import "strings"
import "github.com/franela/goblin"

func Test_BadgeAPI(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("BadgeAPI", func() {
		var api *BadgeAPI

		g.BeforeEach(func() {
			api = &BadgeAPI{}
		})

		g.Describe("parseProfiles", func() {
			g.It("returns an error with invalid profile", func() {
				_, e := api.parseProfiles(strings.NewReader("asdasdasd"))
				g.Assert(e == nil).Equal(false)
			})

			g.It("returns an empty list of profiles if the reader is empty", func() {
				r, e := api.parseProfiles(strings.NewReader(""))
				g.Assert(e).Equal(nil)
				g.Assert(len(r.files)).Equal(0)
			})

			g.It("returns error if line is poorly formatted", func() {
				_, e := api.parseProfiles(strings.NewReader(`mode: atomic
			github.com/dadleyy/marlow/examples/library/models/author.marlow.go:11.81,12.52 3 3
			bad-format
			`))
				g.Assert(e == nil).Equal(false)
			})

			g.It("returns error if line has poor values for numerical info", func() {
				_, e := api.parseProfiles(strings.NewReader(`mode: atomic
			github.com/dadleyy/marlow/examples/library/models/author.marlow.go:11.81,12.52 a basd`))
				g.Assert(e == nil).Equal(false)
			})

			g.It("returns 100%% coverage if that is the case", func() {
				r, e := api.parseProfiles(strings.NewReader(`mode: atomic
			github.com/dadleyy/marlow/examples/library/models/author.marlow.go:11.81,12.52 3 3`))
				g.Assert(e).Equal(nil)
				g.Assert(len(r.files)).Equal(1)
				g.Assert(r.coverage).Equal(100)
			})
		})
	})
}
