package gendry

import "testing"
import "github.com/franela/goblin"

func Test_DisplayAPI(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("DisplayAPI", func() {
		var api *displayAPI

		g.BeforeEach(func() {
			api = &displayAPI{}
		})
	})
}
