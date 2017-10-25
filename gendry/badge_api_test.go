package gendry

import "testing"
import "github.com/franela/goblin"

func Test_BadgeAPI(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("BadgeAPI", func() {
		var api *badgeAPI

		g.BeforeEach(func() {
			api = &badgeAPI{}
		})
	})
}
