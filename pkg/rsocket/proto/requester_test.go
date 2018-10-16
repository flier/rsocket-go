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

		Convey("When send a RequestResponse request", func() {
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
