package proto

import (
	"io"
)

// Responder APIs to handle requests on an RSocket connection.
type Responder interface {
	io.Closer

	// Called when a new `requestResponse` occurs from an Requester.
	HandleRequestResponse(streamId StreamId, request Payload) (Payload, error)

	// Called when a new `requestStream` occurs from an Requester.
	HandleRequestStream(streamId StreamId, request Payload) (PayloadStream, error)

	// Called when a new `requestChannel` occurs from an RSocketRequester.
	HandleRequestChannel(streamId StreamId, payload Payload, requestStream PayloadStream) (PayloadStream, error)

	// Called when a new `fireAndForget` occurs from an RSocketRequester.
	HandleFireAndForget(streamId StreamId, request Payload) error

	// Called when a new `metadataPush` occurs from an RSocketRequester.
	HandleMetadataPush(metadata Metadata) error
}
