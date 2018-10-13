package frame

import (
	"crypto/rand"
	"encoding/binary"
	"io"
)

type Token []byte

const defaultTokenSize = 16
const tokenLenSize = uint16Size

func NewToken() Token {
	token := make([]byte, defaultTokenSize)

	rand.Read(token)

	return token
}

func ReadToken(r io.Reader) (token Token, err error) {
	var len uint16

	if err = binary.Read(r, binary.BigEndian, &len); err != nil {
		return
	}

	return readExact(r, int(len))
}

func (token Token) Size() int {
	return len(token)
}

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
