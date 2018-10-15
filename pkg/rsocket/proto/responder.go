package proto

import (
	"io"
)

// Responder APIs to handle requests on an RSocket connection.
type Responder interface {
	io.Closer

	// Called when a new `requestResponse` occurs from an Requester.
	HandleRequestResponse(streamID StreamID, request Payload) (Payload, error)

	// Called when a new `requestStream` occurs from an Requester.
	HandleRequestStream(streamID StreamID, request Payload) (PayloadStream, error)

	// Called when a new `requestChannel` occurs from an RSocketRequester.
	HandleRequestChannel(streamID StreamID, payload Payload, requestStream PayloadStream) (PayloadStream, error)

	// Called when a new `fireAndForget` occurs from an RSocketRequester.
	HandleFireAndForget(streamID StreamID, request Payload) error

	// Called when a new `metadataPush` occurs from an RSocketRequester.
	HandleMetadataPush(metadata Metadata) error
}
