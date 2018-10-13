package frame

type Type = byte

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
