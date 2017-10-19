package main

import "testing"
import "strings"
import "github.com/franela/goblin"

func Test_parseProfiles(t *testing.T) {
	g := goblin.Goblin(t)

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

	})
}
