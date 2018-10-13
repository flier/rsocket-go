package frame

import (
	"errors"
	"io"
)

var ErrUnknownType = errors.New("unknown frame type")

type Frame interface {
	io.WriterTo

	StreamId() StreamId

	Type() Type

	Flags() Flags

	Size() int
}

func readFrame(r io.Reader, header *Header) (Frame, error) {
	switch header.Type() {
	case TypeSetup:
		return readSetupFrame(r, header)
	case TypeLease:
		return readLeaseFrame(r, header)
	case TypeKeepalive:
		return readKeepaliveFrame(r, header)
	case TypeRequestResponse:
		return readRequestResponseFrame(r, header)
	case TypeRequestFireAndForget:
		return readRequestFireAndForgetFrame(r, header)
	case TypeRequestStream:
		return readRequestStreamFrame(r, header)
	case TypeRequestChannel:
		return readRequestChannelFrame(r, header)
	case TypeRequestN:
		return readRequestNFrame(r, header)
	case TypeCancel:
		return readCancelFrame(r, header)
	case TypePayload:
		return readPayloadFrame(r, header)
	case TypeError:
		return readErrorFrame(r, header)
	case TypeMetadataPush:
		return readMetadataPushFrame(r, header)
	case TypeResume:
		return readResumeFrame(r, header)
	case TypeResumeOk:
		return readResumeOkFrame(r, header)
	case TypeExtension:
		return readExtensionFrame(r, header)
	default:
		return nil, ErrUnknownType
	}
}
