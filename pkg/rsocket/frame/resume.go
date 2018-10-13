package frame

import (
	"encoding/binary"
	"io"
)

const firstAvailableSize = uint64Size

type ResumeFrame struct {
	*Header
	Version        Version
	Token          Token    // Token used for client resume identification.
	LastReceived   Position // The last implied position the client received from the server.
	FirstAvailable Position // The earliest position that the client can rewind back to prior to resending frames.
}

func readResumeFrame(r io.Reader, header *Header) (frame *ResumeFrame, err error) {
	var major, minor uint16
	var resumeToken []byte
	var lastReceived, firstAvailable uint64

	if err = binary.Read(r, binary.BigEndian, &major); err != nil {
		return
	}
	if err = binary.Read(r, binary.BigEndian, &minor); err != nil {
		return
	}

	if resumeToken, err = readToken(r); err != nil {
		return
	}

	if err = binary.Read(r, binary.BigEndian, &lastReceived); err != nil {
		return
	}
	if err = binary.Read(r, binary.BigEndian, &firstAvailable); err != nil {
		return
	}

	frame = &ResumeFrame{
		header,
		Version{major, minor},
		resumeToken,
		Position(lastReceived),
		Position(firstAvailable),
	}

	return
}

func (resume *ResumeFrame) Size() int {
	size := resume.Header.Size() + resume.Version.Size()

	size += tokenLenSize + resume.Token.Size()
	size += lastReceivedSize + firstAvailableSize

	return size
}

func (resume *ResumeFrame) WriteTo(w io.Writer) (wrote int64, err error) {
	if wrote, err = resume.Header.WriteTo(w); err != nil {
		return
	}

	var n int64

	if n, err = resume.Version.WriteTo(w); err != nil {
		return
	}

	wrote += n

	if n, err = resume.Token.WriteTo(w); err != nil {
		return
	}

	wrote += n

	if err = binary.Write(w, binary.BigEndian, uint64(resume.LastReceived)); err != nil {
		return
	}
	if err = binary.Write(w, binary.BigEndian, uint64(resume.FirstAvailable)); err != nil {
		return
	}

	wrote += lastReceivedSize + firstAvailableSize

	return
}

type ResumeOkFrame struct {
	*Header
	LastReceived Position // The last implied position the server received from the client.
}

func readResumeOkFrame(r io.Reader, header *Header) (frame *ResumeOkFrame, err error) {
	var lastReceived uint64

	if err = binary.Read(r, binary.BigEndian, &lastReceived); err != nil {
		return
	}

	frame = &ResumeOkFrame{
		header,
		Position(lastReceived),
	}

	return
}

func (frame *ResumeOkFrame) Size() int {
	return frame.Header.Size() + lastReceivedSize
}

func (frame *ResumeOkFrame) WriteTo(w io.Writer) (wrote int64, err error) {
	var n int64

	if n, err = frame.Header.WriteTo(w); err != nil {
		return
	}

	wrote = n

	if err = binary.Write(w, binary.BigEndian, uint64(frame.LastReceived)); err != nil {
		return
	}

	wrote += lastReceivedSize

	return
}
