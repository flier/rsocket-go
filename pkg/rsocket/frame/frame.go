package frame

import (
	"errors"
	"io"
)

var ErrUnknownType = errors.New("unknown frame type")

type Frame interface {
	io.WriterTo

	Size() int
}

func ReadFrame(r io.Reader, header *Header) (Frame, error) {
	switch header.FrameType {
	case TypeSetup:
		return ReadSetupFrame(r, header)
	case TypeLease:
		return ReadLeaseFrame(r, header)
	case TypeKeepalive:
		return ReadKeepaliveFrame(r, header)
	case TypeRequestResponse:
		return ReadRequestResponseFrame(r, header)
	case TypeRequestFireAndForget:
		return ReadRequestFireAndForgetFrame(r, header)
	case TypeRequestStream:
		return ReadRequestStreamFrame(r, header)
	case TypeRequestChannel:
		return ReadRequestChannelFrame(r, header)
	case TypeRequestN:
		return ReadRequestNFrame(r, header)
	case TypeCancel:
		return ReadCancelFrame(r, header)
	case TypePayload:
		return ReadPayloadFrame(r, header)
	case TypeError:
		return ReadErrorFrame(r, header)
	case TypeMetadataPush:
		return ReadMetadataPushFrame(r, header)
	case TypeResume:
		return ReadResumeFrame(r, header)
	case TypeResumeOk:
		return ReadResumeOkFrame(r, header)
	case TypeExtension:
		return ReadExtensionFrame(r, header)
	default:
		return nil, ErrUnknownType
	}
}
