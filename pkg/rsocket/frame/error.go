package frame

import (
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
)

const errorCodeSize = uint32Size

// ErrorCode is the type of Error.
type ErrorCode uint32

const (
	// ErrReserved is reserved.
	ErrReserved ErrorCode = 0x00000000
	// ErrInvalidSetup indicates the Setup frame is invalid for the serve
	// (it could be that the client is too recent for the old server).
	// Stream ID MUST be 0.
	ErrInvalidSetup ErrorCode = 0x00000001
	// ErrUnsupportedSetup indicates some (or all) of the parameters
	// specified by the client are unsupported by the server.
	// Stream ID MUST be 0.
	ErrUnsupportedSetup ErrorCode = 0x00000002
	// ErrRejectedSetup indicates the server rejected the setup,
	// it can specify the reason in the payload.
	// Stream ID MUST be 0.
	ErrRejectedSetup ErrorCode = 0x00000003
	// ErrRejectedResume indicates the server rejected the resume,
	// it can specify the reason in the payload.
	// Stream ID MUST be 0.
	ErrRejectedResume ErrorCode = 0x00000004
	// ErrConnectionError indicates the connection is being terminated.
	// Stream ID MUST be 0.
	// Sender or Receiver of this frame MAY close the connection immediately
	// without waiting for outstanding streams to terminate.
	ErrConnectionError ErrorCode = 0x00000101
	// ErrApplicationError indicates application layer logic generating error.
	// Stream ID MUST be non-0.
	ErrApplicationError ErrorCode = 0x00000201
	// ErrRejected indicates a valid request, the Responder decided to reject it.
	// The Responder guarantees that it didn't process the request.
	// The reason for the rejection is explained in the metadata section.
	// Stream ID MUST be non-0.
	ErrRejected ErrorCode = 0x00000202
	// ErrCanceled indicates the responder canceled the request
	// but potentially have started processing it
	// (almost identical to REJECTED but doesn't garantee that no side-effect have been started).
	// Stream ID MUST be non-0.
	ErrCanceled ErrorCode = 0x00000203
	// ErrInvalid indicates the request is invalid.
	// Stream ID MUST be non-0.
	ErrInvalid ErrorCode = 0x00000204
	// ErrExtension is reserved for Extension Use.
	ErrExtension ErrorCode = 0xFFFFFFFF
)

func (code ErrorCode) Error() string {
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

func (code ErrorCode) WithMessage(msg string) *Error {
	return &Error{Code: code, Data: msg}
}

// ErrorFrame reports error at connection or application level.
type ErrorFrame struct {
	*Header
	*Error
}

// NewErrorFrame creates a new ErrorFrame.
func NewErrorFrame(streamID StreamID, code ErrorCode, data string) *ErrorFrame {
	return &ErrorFrame{&Header{streamID, TypeError, 0}, &Error{code, data}}
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
		&Error{
			ErrorCode(code),
			string(data),
		},
	}

	return
}

// Error of protocol
type Error struct {
	Code ErrorCode
	Data string
}

func (err *Error) Error() string {
	return fmt.Sprintf("ERROR[%s] %s", err.Code, err.Data)
}

// Err returns the formated error
func (frame *ErrorFrame) Err() error {
	return &Error{frame.Code, frame.Data}
}

// Size returns the encoded size of the frame.
func (frame *ErrorFrame) Size() int {
	return frame.Header.Size() + errorCodeSize + len(frame.Data)
}

// WriteTo writes the encoded frame to w.
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
