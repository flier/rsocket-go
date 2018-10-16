package frame

import (
	"crypto/rand"
	"encoding/binary"
	"io"
)

// Token used for client resume identification
type Token []byte

const defaultTokenSize = 16
const tokenLenSize = uint16Size

// NewToken creates a new randome token.
func NewToken() Token {
	token := make([]byte, defaultTokenSize)

	rand.Read(token)

	return token
}

func readToken(r io.Reader) (token Token, err error) {
	var len uint16

	if err = binary.Read(r, binary.BigEndian, &len); err != nil {
		return
	}

	return readExact(r, int(len))
}

// Size returns the encoded size of the token.
func (token Token) Size() int {
	return len(token)
}

// WriteTo writes the token to w.
func (token Token) WriteTo(w io.Writer) (wrote int64, err error) {
	if err = binary.Write(w, binary.BigEndian, uint16(len(token))); err != nil {
		return
	}

	wrote = tokenLenSize

	var n int64

	if n, err = writeExact(w, token); err != nil {
		return
	}

	wrote += n

	return
}
