package frame

import (
	"fmt"
)

type Type byte

const (
	TypeReserved             Type = iota
	TypeSetup                            // Sent by client to initiate protocol processing.
	TypeLease                            // Sent by Responder to grant the ability to send requests.
	TypeKeepalive                        // Connection keepalive.
	TypeRequestResponse                  // Request single response.
	TypeRequestFireAndForget             // A single one-way message.
	TypeRequestStream                    // Request a completable stream.
	TypeRequestChannel                   // Request a completable stream in both directions.
	TypeRequestN                         // Request N more items with Reactive Streams semantics.
	TypeCancel                           // Cancel outstanding request.
	TypePayload                          // Payload on a stream.
	TypeError                            // Error at connection or application level.
	TypeMetadataPush                     // Asynchronous Metadata frame
	TypeResume                           // Replaces SETUP for Resuming Operation (optional)
	TypeResumeOk                         // Sent in response to a RESUME if resuming operation possible (optional)
	TypeExtension            Type = 0x3F // Used To Extend more frame types as well as extensions.
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
