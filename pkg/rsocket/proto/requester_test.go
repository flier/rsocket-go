package proto

import (
	"context"
	"sync"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/zap"

	"github.com/flier/rsocket-go/pkg/rsocket/frame"
)

func getLogger() (logger *zap.Logger) {
	if testing.Verbose() {
		logger, _ = zap.NewDevelopment()
	} else {
		logger, _ = zap.NewProduction()
	}

	return
}

func TestRequestResponse(t *testing.T) {
	logger := getLogger()
	defer logger.Sync()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	frameSender := make(chan frame.Frame)

	Convey("Given a client requester", t, func() {
		requester := NewRequester(logger, frameSender, ClientStreamIDs(), 16)

		// RQ -> RS: REQUEST_RESPONSE
		// RS -> RQ: PAYLOAD with COMPLETE
		Convey("When send a RequestResponse request for response", func() {
			go func() {
				Convey("The RequestResponse request should be sent", t, func() {
					f := <-frameSender

					So(f, ShouldNotBeNil)
					So(f.StreamID(), ShouldEqual, 1)
					So(f.Type(), ShouldEqual, frame.TypeRequestResponse)
					So(f.Flags(), ShouldEqual, frame.FlagMetadata)

					requestFrame := f.(*frame.RequestResponseFrame)

					So(string(requestFrame.Data), ShouldEqual, "hello")
					So(string(requestFrame.Metadata), ShouldEqual, "world")

					Convey("Then send Payload response", func() {
						payloadFrame := Text("hello world").buildPayloadFrame(f.StreamID(), true)

						So(requester.(*rSocketRequester).handleFrame(ctx, payloadFrame), ShouldBeNil)
					})
				})
			}()

			payload, err := requester.RequestResponse(ctx, Text("hello").WithMetadata([]byte("world")))

			So(payload.Text(), ShouldEqual, "hello world")
			So(err, ShouldBeNil)
		})

		// RQ -> RS: REQUEST_RESPONSE
		// RS -> RQ: ERROR[APPLICATION_ERROR|REJECTED|CANCELED|INVALID]
		Convey("When send a RequestResponse request for error", func() {
			go func() {
				Convey("The RequestResponse request should be sent", t, func() {
					f := <-frameSender

					So(f, ShouldNotBeNil)
					So(f.StreamID(), ShouldEqual, 1)
					So(f.Type(), ShouldEqual, frame.TypeRequestResponse)
					So(f.Flags(), ShouldEqual, 0)

					requestFrame := f.(*frame.RequestResponseFrame)

					So(string(requestFrame.Data), ShouldEqual, "hello")

					Convey("Then send error response", func() {
						errorFrame := frame.NewErrorFrame(f.StreamID(), frame.ErrApplicationError, "for test")

						So(requester.(*rSocketRequester).handleFrame(ctx, errorFrame), ShouldBeNil)
					})
				})
			}()

			payload, err := requester.RequestResponse(ctx, Text("hello"))

			So(payload, ShouldBeNil)
			So(err.Error(), ShouldEqual, "ERROR[APPLICATION_ERROR] for test")
			So(err.(*Error), ShouldResemble, &Error{Code: frame.ErrApplicationError, Data: "for test"})
		})

		// RQ -> RS: REQUEST_RESPONSE
		// RQ -> RS: CANCEL
		Convey("When send a RequestResponse request for cancel", func() {
			go func() {
				Convey("The RequestResponse request should be sent", t, func() {
					f := <-frameSender

					So(f, ShouldNotBeNil)
					So(f.StreamID(), ShouldEqual, 1)
					So(f.Type(), ShouldEqual, frame.TypeRequestResponse)
					So(f.Flags(), ShouldEqual, 0)

					requestFrame := f.(*frame.RequestResponseFrame)

					So(string(requestFrame.Data), ShouldEqual, "hello")

					Convey("Then send cancel response", func() {
						cancelFrame := frame.NewCancelFrame(f.StreamID())

						So(requester.(*rSocketRequester).handleFrame(ctx, cancelFrame), ShouldBeNil)
					})
				})
			}()

			payload, err := requester.RequestResponse(ctx, Text("hello"))

			So(payload, ShouldBeNil)
			So(err, ShouldEqual, context.Canceled)
		})
	})
}

func TestFireAndForget(t *testing.T) {
	logger := getLogger()
	defer logger.Sync()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	frameSender := make(chan frame.Frame)

	Convey("Given a client requester", t, func() {
		requester := NewRequester(logger, frameSender, ClientStreamIDs(), 16)

		Convey("When send a FireAndForget request", func() {
			wg := new(sync.WaitGroup)
			wg.Add(1)

			go func() {
				defer wg.Done()

				Convey("The FireAndForget request should be sent", t, func() {
					f, ok := <-frameSender

					So(ok, ShouldBeTrue)
					So(f, ShouldNotBeNil)
					So(f.StreamID(), ShouldEqual, 1)
					So(f.Type(), ShouldEqual, frame.TypeRequestFireAndForget)
					So(f.Flags(), ShouldEqual, frame.FlagMetadata)

					requestFrame := f.(*frame.RequestFireAndForgetFrame)

					So(string(requestFrame.Data), ShouldEqual, "hello")
					So(string(requestFrame.Metadata), ShouldEqual, "world")
				})
			}()

			So(requester.FireAndForget(ctx, Text("hello").WithMetadata([]byte("world"))), ShouldBeNil)

			wg.Wait()
		})
	})
}

func TestMetadataPush(t *testing.T) {
	logger := getLogger()
	defer logger.Sync()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	frameSender := make(chan frame.Frame)

	Convey("Given a client requester", t, func() {
		requester := NewRequester(logger, frameSender, ClientStreamIDs(), 16)

		Convey("When send a MetadataPush request", func() {
			wg := new(sync.WaitGroup)
			wg.Add(1)

			go func() {
				defer wg.Done()

				Convey("The MetadataPush frame should be sent", t, func() {
					f, ok := <-frameSender

					So(ok, ShouldBeTrue)
					So(f, ShouldNotBeNil)
					So(f.StreamID(), ShouldEqual, 0)
					So(f.Type(), ShouldEqual, frame.TypeMetadataPush)
					So(f.Flags(), ShouldEqual, frame.FlagMetadata)

					frame := f.(*frame.MetadataPushFrame)

					So(string(frame.Metadata), ShouldEqual, "hello")
				})
			}()

			So(requester.MetadataPush(ctx, []byte("hello")), ShouldBeNil)

			wg.Wait()
		})
	})
}
