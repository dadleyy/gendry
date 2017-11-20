package gendry

import "io"
import "bytes"
import "testing"
import "github.com/franela/goblin"

func Test_Tokens(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("newTokenGenerator test suite", func() {
		var s io.Reader

		g.BeforeEach(func() { s = newTokenGenerator(20) })

		g.It("generates a new token", func() {
			out := new(bytes.Buffer)
			io.Copy(out, s)
			g.Assert(len(out.String())).Equal(20)
		})
	})
}
