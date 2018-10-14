package frame

import (
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
)

const errorCodeSize = uint32Size

type ErrorCode uint32

const (
	ErrReserved ErrorCode = 0x00000000
	// The Setup frame is invalid for the server (it could be that the client is
	// too recent for the old server). Stream ID MUST be 0.
	ErrInvalidSetup ErrorCode = 0x00000001
	// Some (or all) of the parameters specified by the client are unsupported by
	// the server. Stream ID MUST be 0.
	ErrUnsupportedSetup ErrorCode = 0x00000002
	// The server rejected the setup, it can specify the reason in the payload.
	// Stream ID MUST be 0.
	ErrRejectedSetup ErrorCode = 0x00000003
	// The server rejected the resume, it can specify the reason in the payload.
	// Stream ID MUST be 0.
	ErrRejectedResume ErrorCode = 0x00000004
	// The connection is being terminated. Stream ID MUST be 0.
	ErrConnectionError ErrorCode = 0x00000101
	// Application layer logic generating a Reactive Streams onError event.
	// Stream ID MUST be non-0.
	ErrApplicationError ErrorCode = 0x00000201
	// Despite being a valid request, the Responder decided to reject it. The
	// Responder guarantees that it didn't process the request. The reason for the
	// rejection is explained in the metadata section. Stream ID MUST be non-0.
	ErrRejected ErrorCode = 0x00000202
	// The responder canceled the request but potentially have started processing
	// it (almost identical to REJECTED but doesn't garantee that no side-effect
	// have been started). Stream ID MUST be non-0.
	ErrCanceled ErrorCode = 0x00000203
	// The request is invalid. Stream ID MUST be non-0.
	ErrInvalid   ErrorCode = 0x00000204
	ErrExtension ErrorCode = 0xFFFFFFFF
)

func (code ErrorCode) String() string {
	switch code {
	case ErrReserved:
		return "RESERVED"
	case ErrInvalidSetup:
		return "INVALID_SETUP"
	case ErrUnsupportedSetup:
		return "UNSUPPORTED_SETUP"
	case ErrRejectedSetup:
		return "REJECTED_SETUP"
	case ErrRejectedResume:
		return "REJECTED_RESUME"
	case ErrConnectionError:
		return "CONNECTION_ERROR"
	case ErrApplicationError:
		return "APPLICATION_ERROR"
	case ErrRejected:
		return "REJECTED"
	case ErrCanceled:
		return "CANCELED"
	case ErrInvalid:
		return "INVALID"
	case ErrExtension:
		return "EXT"
	default:
		return fmt.Sprintf("%08x", uint32(code))
	}
}

type ErrorFrame struct {
	*Header
	Code ErrorCode
	Data string
}

func readErrorFrame(r io.Reader, header *Header) (frame *ErrorFrame, err error) {
	var code uint32
	var data []byte

	if err = binary.Read(r, binary.BigEndian, &code); err != nil {
		return
	}

	if data, err = ioutil.ReadAll(r); err != nil {
		return
	}

	frame = &ErrorFrame{
		header,
		ErrorCode(code),
		string(data),
	}

	return
}

func (err *ErrorFrame) Err() error {
	return fmt.Errorf("ERROR[%s] %s", err.Code, err.Data)
}

func (frame *ErrorFrame) Size() int {
	return frame.Header.Size() + errorCodeSize + len(frame.Data)
}

func (frame *ErrorFrame) WriteTo(w io.Writer) (wrote int64, err error) {
	var n int64

	if n, err = frame.Header.WriteTo(w); err != nil {
		return
	}

	wrote = n

	if err = binary.Write(w, binary.BigEndian, uint32(frame.Code)); err != nil {
		return
	}

	wrote += errorCodeSize

	if n, err = writeExact(w, []byte(frame.Data)); err != nil {
		return
	}

	wrote += n

	return
}
