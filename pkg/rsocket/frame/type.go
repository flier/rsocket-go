package frame

import (
	"fmt"
)

// Type of frame
type Type byte

const (
	// TypeReserved is Reserved
	TypeReserved Type = iota
	// TypeSetup sent by client to initiate protocol processing.
	TypeSetup
	// TypeLease sent by Responder to grant the ability to send requests.
	TypeLease
	// TypeKeepalive sent to make connection keepalive.
	TypeKeepalive
	// TypeRequestResponse sent by client to request a single response.
	TypeRequestResponse
	// TypeRequestFireAndForget sent by client to request a single one-way message.
	TypeRequestFireAndForget
	// TypeRequestStream sent by client to request a completable stream.
	TypeRequestStream
	// TypeRequestChannel sent by client to request a completable stream in both directions.
	TypeRequestChannel
	// TypeRequestN sent to request N more items with Reactive Streams semantics.
	TypeRequestN
	// TypeCancel sent to cancel outstanding request.
	TypeCancel
	// TypePayload send payload on a stream.
	TypePayload
	// TypeError send error at connection or application level.
	TypeError
	// TypeMetadataPush send a metadata.
	TypeMetadataPush
	// TypeResume replaces SETUP for Resuming Operation (optional)
	TypeResume
	// TypeResumeOk sent in response to a RESUME if resuming operation possible (optional)
	TypeResumeOk
	// TypeExtension used To Extend more frame types as well as extensions.
	TypeExtension Type = 0x3F
)

func (t Type) String() string {
	switch t {
	case TypeReserved:
		return "RESERVED"
	case TypeSetup:
		return "SETUP"
	case TypeLease:
		return "LEASE"
	case TypeKeepalive:
		return "KEEPALIVE"
	case TypeRequestResponse:
		return "REQUEST_RESPONSE"
	case TypeRequestFireAndForget:
		return "REQUEST_FNF"
	case TypeRequestStream:
		return "REQUEST_STREAM"
	case TypeRequestChannel:
		return "REQUEST_CHANNEL"
	case TypeRequestN:
		return "REQUEST_N"
	case TypeCancel:
		return "CANCEL"
	case TypePayload:
		return "PAYLOAD"
	case TypeError:
		return "ERROR"
	case TypeMetadataPush:
		return "METADATA_PUSH"
	case TypeResume:
		return "RESUME"
	case TypeResumeOk:
		return "RESUME_OK"
	case TypeExtension:
		return "EXT"
	default:
		return fmt.Sprintf("TYPE[%d]", uint8(t))
	}
}
