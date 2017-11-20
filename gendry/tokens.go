package gendry

import "io"
import "strings"
import "crypto/rand"
import "encoding/hex"

func newTokenGenerator(size int) io.Reader {
	pr, pw := io.Pipe()

	go func() {
		raw := make([]byte, size/2)
		rand.Read(raw)
		buffer := strings.NewReader(hex.EncodeToString(raw))
		io.Copy(pw, buffer)
		pw.CloseWithError(nil)
	}()

	return pr
}
