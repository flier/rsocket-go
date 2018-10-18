package proto

import (
	"context"
	"flag"
	"os"
	"sync"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/zap"

	"github.com/flier/rsocket-go/pkg/rsocket/frame"
)

const initReqs = 16

var logger *zap.Logger

func TestMain(m *testing.M) {
	flag.Parse()

	if testing.Verbose() && !testing.Short() {
		logger, _ = zap.NewDevelopment()
	} else if testing.Short() {
		logger, _ = zap.NewProduction()
	} else {
		logger = zap.NewNop()
	}

	defer logger.Sync()

	os.Exit(m.Run())
}

type testEnv struct {
	t         *testing.T
	ctx       context.Context
	cancel    context.CancelFunc
	requests  frameChan
	responses frameChan
	requester *rSocketRequester
}

func (env *testEnv) Serv(wg *sync.WaitGroup) {
	wg.Add(1)

	go func() error {
		defer wg.Done()

		for {
			f, err := env.responses.Recv(env.ctx)

			if err != nil {
				return err
			}

			if f == nil {
				return nil
			}

			err = env.requester.handleFrame(env.ctx, f)

			if err != nil {
				return err
			}

			env.t.Logf("RS -> RQ: %s", f)
		}
	}()
}

type runCallback func(env *testEnv)

func asClient(callback func(ctx context.Context, requester *rSocketRequester)) runCallback {
	return func(env *testEnv) {
		Convey("RQ -> Given a client requester", env.t, func() {
			callback(env.ctx, env.requester)
		})
	}
}

func asServer(callback func(ctx context.Context, cancel context.CancelFunc, requests FrameReceiver, responses FrameSender)) runCallback {
	return func(env *testEnv) {
		defer env.responses.Close()

		Convey("RS -> Given a server responder", env.t, func() {
			callback(env.ctx, env.cancel, env.requests, env.responses)
		})
	}
}

type logFrameSender struct {
	t *testing.T
	c frameChan
}

func (sender logFrameSender) Close() error {
	close(sender.c)

	return nil
}

func (sender logFrameSender) Send(ctx context.Context, f frame.Frame) error {
	sender.t.Logf("RQ -> RS: %s", f)

	return sender.c.Send(ctx, f)
}

func run(t *testing.T, callbacks ...runCallback) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	requests := make(frameChan)
	responses := make(frameChan)
	requester := NewRequester(logger, logFrameSender{t, requests}, ClientStreamIDs(), uint(initReqs)).(*rSocketRequester)

	wg := new(sync.WaitGroup)
	defer wg.Wait()

	env := testEnv{t, ctx, cancel, requests, responses, requester}
	env.Serv(wg)

	for _, callback := range callbacks {
		callback := callback

		go func() {
			defer wg.Done()

			callback(&env)
		}()
	}
	wg.Add(len(callbacks))
}

func checkFrameHeader(f frame.Frame, streamID StreamID, tp frame.Type, flags frame.Flags) {
	So(f, ShouldNotBeNil)
	So(f.StreamID(), ShouldEqual, streamID)
	So(f.Type(), ShouldEqual, tp)
	So(f.Flags(), ShouldEqual, flags)
}

// RQ -> RS: REQUEST_STREAM
// RS -> RQ: PAYLOAD*
// RS -> RQ: COMPLETE
func TestRequestStreamComplete(t *testing.T) {
	run(t,
		asClient(func(ctx context.Context, requester *rSocketRequester) {
			Convey("RQ -> RS: When request stream for payloads", func() {
				responses, err := requester.RequestStream(ctx, Text("hello").WithMetadata([]byte("world")))
				So(err, ShouldBeNil)

				Convey("RQ -> RS: Then payload stream should be ready", func() {
					payload, _ := responses.Recv(ctx)
					So(payload, ShouldResemble, Text("foo"))

					payload, _ = responses.Recv(ctx)
					So(payload, ShouldResemble, Text("bar"))

					payload, _ = responses.Recv(ctx)
					So(payload, ShouldBeNil)
				})
			})
		}),
		asServer(func(ctx context.Context, cancel context.CancelFunc, requests FrameReceiver, responses FrameSender) {
			Convey("RS -> RQ: Then request should be sent", func() {
				f, err := requests.Recv(ctx)

				So(err, ShouldBeNil)
				checkFrameHeader(f, 1, frame.TypeRequestStream, frame.FlagMetadata)

				requestFrame := f.(*frame.RequestStreamFrame)

				So(requestFrame, ShouldNotBeNil)
				So(requestFrame.InitialRequests, ShouldEqual, initReqs)
				So(string(requestFrame.Data), ShouldEqual, "hello")
				So(string(requestFrame.Metadata), ShouldEqual, "world")

				Convey("RS -> RQ: Then send payload", func() {
					payloadFrame := buildPayloadFrame(f.StreamID(), false, Text("foo"))
					So(responses.Send(ctx, payloadFrame), ShouldBeNil)

					payloadFrame = buildPayloadFrame(f.StreamID(), true, Text("bar"))
					So(responses.Send(ctx, payloadFrame), ShouldBeNil)
				})
			})
		}),
	)
}

// RQ -> RS: REQUEST_STREAM
// RS -> RQ: PAYLOAD*
// RS -> RQ: ERROR[APPLICATION_ERROR|REJECTED|CANCELED|INVALID]
func TestRequestStreamWithError(t *testing.T) {
	run(t,
		asClient(func(ctx context.Context, requester *rSocketRequester) {
			Convey("RQ -> RS: Then request stream for payloads", func() {
				responses, err := requester.RequestStream(ctx, Text("hello").WithMetadata([]byte("world")))
				So(err, ShouldBeNil)

				Convey("RQ -> RS: Then response stream should be ready", func() {
					payload, err := responses.Recv(ctx)
					So(payload, ShouldResemble, Text("foo"))

					payload, err = responses.Recv(ctx)
					So(payload, ShouldResemble, Text("bar"))

					payload, err = responses.Recv(ctx)
					So(err, ShouldResemble, frame.ErrApplicationError.WithMessage("for test"))

					payload, err = responses.Recv(ctx)
					So(payload, ShouldBeNil)
				})
			})
		}),
		asServer(func(ctx context.Context, cancel context.CancelFunc, requests FrameReceiver, responses FrameSender) {
			Convey("RS -> RQ: Then request should be sent", func() {
				f, err := requests.Recv(ctx)

				So(err, ShouldBeNil)
				checkFrameHeader(f, 1, frame.TypeRequestStream, frame.FlagMetadata)

				requestFrame := f.(*frame.RequestStreamFrame)

				So(requestFrame, ShouldNotBeNil)
				So(requestFrame.InitialRequests, ShouldEqual, initReqs)
				So(string(requestFrame.Data), ShouldEqual, "hello")
				So(string(requestFrame.Metadata), ShouldEqual, "world")

				Convey("RS -> RQ: Then send payload with error", func() {
					payloadFrame := buildPayloadFrame(f.StreamID(), false, Text("foo"))
					So(responses.Send(ctx, payloadFrame), ShouldBeNil)

					payloadFrame = buildPayloadFrame(f.StreamID(), false, Text("bar"))
					So(responses.Send(ctx, payloadFrame), ShouldBeNil)

					errorFrame := frame.NewErrorFrame(f.StreamID(), frame.ErrApplicationError, "for test")
					So(responses.Send(ctx, errorFrame), ShouldBeNil)
				})
			})
		}),
	)
}

// RQ -> RS: REQUEST_STREAM
// RS -> RQ: PAYLOAD*
// RQ -> RS: CANCEL
func TestRequestStreamCanceled(t *testing.T) {
	run(t,
		asClient(func(ctx context.Context, requester *rSocketRequester) {
			Convey("When request stream for payloads with cancel", func() {
				responses, err := requester.RequestStream(ctx, Text("hello").WithMetadata([]byte("world")))

				So(err, ShouldBeNil)

				Convey("Then payload stream should be ready", func() {
					payload, err := responses.Recv(ctx)
					So(payload, ShouldResemble, Text("foo"))

					payload, err = responses.Recv(ctx)
					So(payload, ShouldResemble, Text("bar"))

					payload, err = responses.Recv(ctx)
					So(err, ShouldEqual, context.Canceled)

					payload, err = responses.Recv(ctx)
					So(payload, ShouldBeNil)
				})
			})
		}),
		asServer(func(ctx context.Context, cancel context.CancelFunc, requests FrameReceiver, responses FrameSender) {
			Convey("Then request should be sent", func() {
				f, err := requests.Recv(ctx)

				So(err, ShouldBeNil)
				checkFrameHeader(f, 1, frame.TypeRequestStream, frame.FlagMetadata)

				requestFrame := f.(*frame.RequestStreamFrame)

				So(requestFrame, ShouldNotBeNil)
				So(requestFrame.InitialRequests, ShouldEqual, initReqs)
				So(string(requestFrame.Data), ShouldEqual, "hello")
				So(string(requestFrame.Metadata), ShouldEqual, "world")

				Convey("Then send payload with error", func() {
					payloadFrame := buildPayloadFrame(f.StreamID(), false, Text("foo"))
					So(responses.Send(ctx, payloadFrame), ShouldBeNil)

					payloadFrame = buildPayloadFrame(f.StreamID(), false, Text("bar"))
					So(responses.Send(ctx, payloadFrame), ShouldBeNil)

					cancelFrame := frame.NewCancelFrame(f.StreamID())
					So(responses.Send(ctx, cancelFrame), ShouldBeNil)
				})
			})
		}),
	)
}

// RQ -> RS: REQUEST_CHANNEL
// RQ -> RS: PAYLOAD*
// RQ -> RS: COMPLETE
//
// intermixed with
//
// RS -> RQ: PAYLOAD*
// RS -> RQ: COMPLETE
func TestRequestChannelCompleteFromRequesterAndResponder(t *testing.T) {
	run(t,
		asClient(func(ctx context.Context, requester *rSocketRequester) {
			Convey("RQ -> RS: When payloads be ready before send request", func() {
				requests := make(chan *Result, 16)
				sink := &PayloadSink{requests}

				So(sink.Send(ctx, Ok(Text("hello"))), ShouldBeNil)
				So(sink.Send(ctx, Ok(Text("world"))), ShouldBeNil)
				So(sink.Close(), ShouldBeNil)

				responses, err := requester.RequestChannel(ctx, &PayloadStream{requests})

				So(err, ShouldBeNil)

				Convey("RQ -> RS: Then payload stream should be ready", func() {
					payload, _ := responses.Recv(ctx)
					So(payload, ShouldResemble, Text("foo"))

					payload, _ = responses.Recv(ctx)
					So(payload, ShouldResemble, Text("bar"))

					payload, _ = responses.Recv(ctx)
					So(payload, ShouldBeNil)
				})
			})
		}),
		asServer(func(ctx context.Context, cancel context.CancelFunc, requests FrameReceiver, responses FrameSender) {
			Convey("RS -> RQ: Then request should be ready", func() {
				f, err := requests.Recv(ctx)
				So(err, ShouldBeNil)
				checkFrameHeader(f, 1, frame.TypeRequestChannel, 0)

				Convey("RS -> RQ: Then payload should be embedded", func() {
					requestFrame := f.(*frame.RequestChannelFrame)
					So(requestFrame, ShouldNotBeNil)
					So(requestFrame.InitialRequests, ShouldEqual, initReqs)
					So(string(requestFrame.Data), ShouldEqual, "hello")

					Convey("RS -> RQ: Then send requestN back to requester", func() {
						requestNFrame := frame.NewRequestNFrame(f.StreamID(), uint32(initReqs))
						So(responses.Send(ctx, requestNFrame), ShouldBeNil)

						Convey("RS -> RQ: Then payload should be sent", func() {
							f, err := requests.Recv(ctx)
							So(err, ShouldBeNil)
							checkFrameHeader(f, 1, frame.TypePayload, frame.FlagNext)

							payloadFrame := f.(*frame.PayloadFrame)

							So(payloadFrame, ShouldNotBeNil)
							So(string(payloadFrame.Data), ShouldEqual, "world")

							Convey("RS -> RQ: Then requests stream should be complete", func() {
								f, err := requests.Recv(ctx)
								So(err, ShouldBeNil)
								checkFrameHeader(f, 1, frame.TypePayload, frame.FlagComplete)

								payloadFrame := f.(*frame.PayloadFrame)
								So(payloadFrame, ShouldNotBeNil)
								So(payloadFrame.Data, ShouldBeNil)

								Convey("RS -> RQ: Then send payload", func() {
									payloadFrame := buildPayloadFrame(f.StreamID(), false, Text("foo"))
									So(responses.Send(ctx, payloadFrame), ShouldBeNil)

									payloadFrame = buildPayloadFrame(f.StreamID(), true, Text("bar"))
									So(responses.Send(ctx, payloadFrame), ShouldBeNil)
								})
							})
						})
					})
				})
			})
		}),
	)
}

// RQ -> RS: REQUEST_CHANNEL
// RQ -> RS: PAYLOAD*
// RQ -> RS: ERROR[APPLICATION_ERROR]
//
// intermixed with
//
// RS -> RQ: PAYLOAD*
func TestRequestChannelErrorFromRequesterAndResponderTerminates(t *testing.T) {
	run(t,
		asClient(func(ctx context.Context, requester *rSocketRequester) {
			requests := make(chan *Result, 128)

			Convey("Then send request immediately", func() {
				responses, err := requester.RequestChannel(ctx, &PayloadStream{requests})

				So(err, ShouldBeNil)
				Convey("When payloads sent after request", func() {
					sink := &PayloadSink{requests}

					So(sink.Send(ctx, Ok(Text("hello"))), ShouldBeNil)

					Convey("Then payload stream should be ready", func() {
						payload, _ := responses.Recv(ctx)
						So(payload, ShouldResemble, Text("world"))

						Convey("Then send error", func() {
							So(sink.Send(ctx, Err(frame.ErrApplicationError.WithMessage("for test"))), ShouldBeNil)
							So(sink.Close(), ShouldBeNil)

							Convey("Then the reciever should be close", func() {
								payload, _ = responses.Recv(ctx)
								So(payload, ShouldBeNil)
							})
						})
					})
				})
			})
		}),
		asServer(func(ctx context.Context, cancel context.CancelFunc, requests FrameReceiver, responses FrameSender) {
			Convey("Then channel request should be ready", func() {
				f, err := requests.Recv(ctx)
				So(err, ShouldBeNil)
				checkFrameHeader(f, 1, frame.TypeRequestChannel, 0)

				Convey("Then request payload should be nil", func() {
					requestFrame := f.(*frame.RequestChannelFrame)

					So(requestFrame, ShouldNotBeNil)
					So(requestFrame.InitialRequests, ShouldEqual, initReqs)
					So(requestFrame.Data, ShouldBeNil)

					Convey("Then send requestN back to requester", func() {
						requestNFrame := frame.NewRequestNFrame(f.StreamID(), uint32(initReqs))

						So(responses.Send(ctx, requestNFrame), ShouldBeNil)

						Convey("Then payload should be sent", func() {
							f, err := requests.Recv(ctx)
							So(err, ShouldBeNil)
							checkFrameHeader(f, 1, frame.TypePayload, frame.FlagNext)

							payloadFrame := f.(*frame.PayloadFrame)

							So(payloadFrame, ShouldNotBeNil)
							So(string(payloadFrame.Data), ShouldEqual, "hello")

							Convey("Then send response", func() {
								payloadFrame := buildPayloadFrame(f.StreamID(), false, Text("world"))

								So(responses.Send(ctx, payloadFrame), ShouldBeNil)

								Convey("Then error should be sent", func() {
									f, err := requests.Recv(ctx)
									So(err, ShouldBeNil)
									checkFrameHeader(f, 1, frame.TypeError, 0)

									errorFrame := f.(*frame.ErrorFrame)

									So(errorFrame, ShouldNotBeNil)
									So(errorFrame.Code, ShouldEqual, frame.ErrApplicationError)
									So(errorFrame.Data, ShouldEqual, "for test")
								})
							})
						})
					})
				})
			})
		}),
	)
}

// RQ -> RS: REQUEST_CHANNEL
// RQ -> RS: PAYLOAD*
// RQ -> RS: ERROR[APPLICATION_ERROR]
//
// intermixed with
//
// RS -> RQ: PAYLOAD*
// RS -> RQ: COMPLETE
func TestRequestChannelErrorFromRequesterAndResponderAlreadyCompleted(t *testing.T) {
	run(t,
		asClient(func(ctx context.Context, requester *rSocketRequester) {
			requests := make(chan *Result, 128)

			Convey("RQ -> RS: Then send request immediately", func() {
				responses, err := requester.RequestChannel(ctx, &PayloadStream{requests})
				So(err, ShouldBeNil)

				Convey("RQ -> RS: When payloads sent after request", func() {
					sink := &PayloadSink{requests}

					So(sink.Send(ctx, Ok(Text("hello"))), ShouldBeNil)

					Convey("RQ -> RS: Then payload stream should be ready", func() {
						payload, _ := responses.Recv(ctx)
						So(payload, ShouldResemble, Text("world"))

						Convey("RQ -> RS: Then send error", func() {
							So(sink.Send(ctx, Err(frame.ErrApplicationError.WithMessage("for test"))), ShouldBeNil)
							So(sink.Close(), ShouldBeNil)

							Convey("RQ -> RS: Then the reciever should be close", func() {
								payload, _ = responses.Recv(ctx)
								So(payload, ShouldBeNil)
							})
						})
					})
				})
			})
		}),
		asServer(func(ctx context.Context, cancel context.CancelFunc, requests FrameReceiver, responses FrameSender) {
			Convey("RS -> RQ: Then channel request should be ready", func() {
				f, err := requests.Recv(ctx)
				So(err, ShouldBeNil)
				checkFrameHeader(f, 1, frame.TypeRequestChannel, 0)

				Convey("RS -> RQ: Then request payload should be nil", func() {
					requestFrame := f.(*frame.RequestChannelFrame)

					So(requestFrame, ShouldNotBeNil)
					So(requestFrame.InitialRequests, ShouldEqual, initReqs)
					So(requestFrame.Data, ShouldBeNil)

					Convey("RS -> RQ: Then send requestN back to requester", func() {
						requestNFrame := frame.NewRequestNFrame(f.StreamID(), uint32(initReqs))

						So(responses.Send(ctx, requestNFrame), ShouldBeNil)

						Convey("RS -> RQ: Then payload should be sent", func() {
							f, err := requests.Recv(ctx)
							So(err, ShouldBeNil)
							checkFrameHeader(f, 1, frame.TypePayload, frame.FlagNext)

							payloadFrame := f.(*frame.PayloadFrame)

							So(payloadFrame, ShouldNotBeNil)
							So(string(payloadFrame.Data), ShouldEqual, "hello")

							Convey("RS -> RQ: Then send response", func() {
								payloadFrame := buildPayloadFrame(f.StreamID(), true, Text("world"))

								So(responses.Send(ctx, payloadFrame), ShouldBeNil)

								Convey("RS -> RQ: Then error should be sent", func() {
									f, err := requests.Recv(ctx)
									So(err, ShouldBeNil)
									checkFrameHeader(f, 1, frame.TypeError, 0)

									errorFrame := f.(*frame.ErrorFrame)

									So(errorFrame, ShouldNotBeNil)
									So(errorFrame.Code, ShouldEqual, frame.ErrApplicationError)
									So(string(errorFrame.Data), ShouldEqual, "for test")
								})
							})
						})
					})
				})
			})
		}),
	)
}

// RQ -> RS: REQUEST_CHANNEL
// RQ -> RS: PAYLOAD*
//
// intermixed with
//
// RS -> RQ: PAYLOAD*
// RS -> RQ: ERROR[APPLICATION_ERROR|REJECTED|CANCELED|INVALID]
func TestRequestChannelErrorFromResponderAndRequesterTerminates(t *testing.T) {
	run(t,
		asClient(func(ctx context.Context, requester *rSocketRequester) {
			requests := make(chan *Result, 128)

			Convey("RQ -> RS: Then send request immediately", func() {
				responses, err := requester.RequestChannel(ctx, &PayloadStream{requests})
				So(err, ShouldBeNil)

				Convey("RQ -> When payloads sent after request", func() {
					sink := &PayloadSink{requests}

					So(sink.Send(ctx, Ok(Text("hello"))), ShouldBeNil)

					Convey("RQ -> RS: Then payload stream should be ready", func() {
						payload, err := responses.Recv(ctx)
						So(payload, ShouldResemble, Text("world"))

						Convey("RQ -> RS: Then recive error", func() {
							payload, err = responses.Recv(ctx)
							So(err, ShouldResemble, frame.ErrApplicationError.WithMessage("for test"))

							Convey("RQ -> RS: Then the send payload should be fail", func() {
								So(sink.Send(ctx, Ok(Text("world"))), ShouldBeNil)
							})
						})
					})
				})
			})
		}),
		asServer(func(ctx context.Context, cancel context.CancelFunc, requests FrameReceiver, responses FrameSender) {
			Convey("RS -> RQ: Then channel request should be ready", func() {
				f, err := requests.Recv(ctx)
				So(err, ShouldBeNil)
				checkFrameHeader(f, 1, frame.TypeRequestChannel, 0)

				Convey("RS -> RQ: Then request payload should be nil", func() {
					requestFrame := f.(*frame.RequestChannelFrame)

					So(requestFrame, ShouldNotBeNil)
					So(requestFrame.InitialRequests, ShouldEqual, initReqs)
					So(requestFrame.Data, ShouldBeNil)

					Convey("RS -> RQ: Then send requestN back to requester", func() {
						requestNFrame := frame.NewRequestNFrame(f.StreamID(), uint32(initReqs))

						So(responses.Send(ctx, requestNFrame), ShouldBeNil)

						Convey("RS -> RQ: Then payload should be sent", func() {
							f, err := requests.Recv(ctx)
							So(err, ShouldBeNil)
							checkFrameHeader(f, 1, frame.TypePayload, frame.FlagNext)

							payloadFrame := f.(*frame.PayloadFrame)

							So(payloadFrame, ShouldNotBeNil)
							So(string(payloadFrame.Data), ShouldEqual, "hello")

							Convey("RS -> RQ: Then send response and error", func() {
								payloadFrame := buildPayloadFrame(f.StreamID(), false, Text("world"))
								So(responses.Send(ctx, payloadFrame), ShouldBeNil)

								errorFrame := frame.NewErrorFrame(f.StreamID(), frame.ErrApplicationError, "for test")
								So(responses.Send(ctx, errorFrame), ShouldBeNil)
							})
						})
					})
				})
			})
		}),
	)
}

// RQ -> RS: REQUEST_CHANNEL
// RQ -> RS: PAYLOAD*
// RQ -> RS: COMPLETE
//
// intermixed with
//
// RS -> RQ: PAYLOAD*
// RS -> RQ: ERROR[APPLICATION_ERROR|REJECTED|CANCELED|INVALID]
func TestRequestChannelErrorFromResponderAndRequesterAlreadyCompleted(t *testing.T) {
	run(t,
		asClient(func(ctx context.Context, requester *rSocketRequester) {
			Convey("RQ -> RS: When payloads be ready before send request", func() {
				requests := make(chan *Result, 16)
				sink := &PayloadSink{requests}

				So(sink.Send(ctx, Ok(Text("hello"))), ShouldBeNil)
				So(sink.Send(ctx, Ok(Text("world"))), ShouldBeNil)
				So(sink.Close(), ShouldBeNil)

				responses, err := requester.RequestChannel(ctx, &PayloadStream{requests})

				So(err, ShouldBeNil)

				Convey("RQ -> RS: Then payload stream should be ready", func() {
					payload, _ := responses.Recv(ctx)
					So(payload, ShouldResemble, Text("foo"))

					_, err = responses.Recv(ctx)
					So(err, ShouldResemble, frame.ErrApplicationError.WithMessage("for test"))

					payload, _ = responses.Recv(ctx)
					So(payload, ShouldBeNil)
				})
			})
		}),
		asServer(func(ctx context.Context, cancel context.CancelFunc, requests FrameReceiver, responses FrameSender) {
			Convey("RS -> RQ: Then channel request should be ready", func() {
				f, err := requests.Recv(ctx)
				So(err, ShouldBeNil)
				checkFrameHeader(f, 1, frame.TypeRequestChannel, 0)

				Convey("RS -> RQ: Then request payload should be nil", func() {
					requestFrame := f.(*frame.RequestChannelFrame)

					So(requestFrame, ShouldNotBeNil)
					So(requestFrame.InitialRequests, ShouldEqual, initReqs)
					So(string(requestFrame.Data), ShouldEqual, "hello")

					Convey("RS -> RQ: Then send requestN back to requester", func() {
						requestNFrame := frame.NewRequestNFrame(f.StreamID(), uint32(initReqs))

						So(responses.Send(ctx, requestNFrame), ShouldBeNil)

						Convey("RS -> RQ: Then payload should be sent", func() {
							f, err := requests.Recv(ctx)
							So(err, ShouldBeNil)
							checkFrameHeader(f, 1, frame.TypePayload, frame.FlagNext)

							payloadFrame := f.(*frame.PayloadFrame)
							So(payloadFrame, ShouldNotBeNil)
							So(string(payloadFrame.Data), ShouldEqual, "world")

							Convey("RS -> RQ: Then send response and error", func() {
								payloadFrame := buildPayloadFrame(f.StreamID(), false, Text("foo"))
								So(responses.Send(ctx, payloadFrame), ShouldBeNil)

								errorFrame := frame.NewErrorFrame(f.StreamID(), frame.ErrApplicationError, "for test")
								So(responses.Send(ctx, errorFrame), ShouldBeNil)
							})
						})
					})
				})
			})
		}),
	)
}

// RQ -> RS: REQUEST_CHANNEL
// RQ -> RS: PAYLOAD*
// RQ -> RS: COMPLETE
// RQ -> RS: CANCEL
//
// intermixed with
//
// RS -> RQ: PAYLOAD*
func TestRequestChannelCancelFromRequesterAndResponderTerminates(t *testing.T) {
	run(t,
		asClient(func(ctx context.Context, requester *rSocketRequester) {
			Convey("RQ -> RS: When payloads be ready before send request", func() {
				requests := make(chan *Result, 16)
				sink := &PayloadSink{requests}

				So(sink.Send(ctx, Ok(Text("hello"))), ShouldBeNil)
				So(sink.Send(ctx, Ok(Text("world"))), ShouldBeNil)
				So(sink.Send(ctx, Err(context.Canceled)), ShouldBeNil)
				So(sink.Close(), ShouldBeNil)

				responses, err := requester.RequestChannel(ctx, &PayloadStream{requests})

				So(err, ShouldBeNil)

				Convey("RQ -> RS: Then payload stream should be ready", func() {
					payload, _ := responses.Recv(ctx)
					So(payload, ShouldResemble, Text("foo"))

					payload, _ = responses.Recv(ctx)
					So(payload, ShouldBeNil)
				})
			})
		}),
		asServer(func(ctx context.Context, cancel context.CancelFunc, requests FrameReceiver, responses FrameSender) {
			Convey("RS -> RQ: Then channel request should be ready", func() {
				f, err := requests.Recv(ctx)
				So(err, ShouldBeNil)
				checkFrameHeader(f, 1, frame.TypeRequestChannel, 0)

				Convey("RS -> RQ: Then request payload should be nil", func() {
					requestFrame := f.(*frame.RequestChannelFrame)

					So(requestFrame, ShouldNotBeNil)
					So(requestFrame.InitialRequests, ShouldEqual, initReqs)
					So(string(requestFrame.Data), ShouldEqual, "hello")

					Convey("RS -> RQ: Then send requestN back to requester", func() {
						requestNFrame := frame.NewRequestNFrame(f.StreamID(), uint32(initReqs))
						So(responses.Send(ctx, requestNFrame), ShouldBeNil)

						Convey("RS -> RQ: Then payload should be sent", func() {
							f, err := requests.Recv(ctx)
							So(err, ShouldBeNil)
							checkFrameHeader(f, 1, frame.TypePayload, frame.FlagNext)

							payloadFrame := f.(*frame.PayloadFrame)
							So(payloadFrame, ShouldNotBeNil)
							So(string(payloadFrame.Data), ShouldEqual, "world")

							Convey("RS -> RQ: Then send response and error", func() {
								payloadFrame := buildPayloadFrame(f.StreamID(), false, Text("foo"))
								So(responses.Send(ctx, payloadFrame), ShouldBeNil)

								f, err := requests.Recv(ctx)
								So(err, ShouldBeNil)
								checkFrameHeader(f, 1, frame.TypeCancel, 0)
							})
						})
					})
				})
			})
		}),
	)
}

// RQ -> RS: REQUEST_RESPONSE
// RS -> RQ: PAYLOAD with COMPLETE
func TestRequestResponseComplete(t *testing.T) {
	run(t,
		asClient(func(ctx context.Context, requester *rSocketRequester) {
			Convey("When request for response", func() {
				payload, err := requester.RequestResponse(ctx, Text("hello").WithMetadata([]byte("world")))

				So(err, ShouldBeNil)
				So(payload.Text(), ShouldEqual, "hello world")
			})
		}),
		asServer(func(ctx context.Context, cancel context.CancelFunc, requests FrameReceiver, responses FrameSender) {
			Convey("The request should be sent", func() {
				f, err := requests.Recv(ctx)

				So(err, ShouldBeNil)
				checkFrameHeader(f, 1, frame.TypeRequestResponse, frame.FlagMetadata)

				requestFrame := f.(*frame.RequestResponseFrame)

				So(requestFrame, ShouldNotBeNil)
				So(string(requestFrame.Data), ShouldEqual, "hello")
				So(string(requestFrame.Metadata), ShouldEqual, "world")

				Convey("Then send payload", func() {
					payloadFrame := buildPayloadFrame(f.StreamID(), true, Text("hello world"))

					So(responses.Send(ctx, payloadFrame), ShouldBeNil)
				})
			})
		}),
	)
}

// RQ -> RS: REQUEST_RESPONSE
// RS -> RQ: ERROR[APPLICATION_ERROR|REJECTED|CANCELED|INVALID]
func TestRequestResponseWithError(t *testing.T) {
	run(t,
		asClient(func(ctx context.Context, requester *rSocketRequester) {
			Convey("RQ -> When request for response with error", func() {
				payload, err := requester.RequestResponse(ctx, Text("hello"))
				So(payload, ShouldBeNil)

				So(err.Error(), ShouldEqual, "ERROR[APPLICATION_ERROR] for test")
				So(err.(*Error), ShouldResemble, frame.ErrApplicationError.WithMessage("for test"))
			})
		}),
		asServer(func(ctx context.Context, cancel context.CancelFunc, requests FrameReceiver, responses FrameSender) {
			Convey("The request should be sent", func() {
				f, err := requests.Recv(ctx)
				So(err, ShouldBeNil)
				checkFrameHeader(f, 1, frame.TypeRequestResponse, 0)

				requestFrame := f.(*frame.RequestResponseFrame)
				So(requestFrame, ShouldNotBeNil)
				So(string(requestFrame.Data), ShouldEqual, "hello")

				Convey("RS ->Then send error", func() {
					errorFrame := frame.NewErrorFrame(f.StreamID(), frame.ErrApplicationError, "for test")
					So(responses.Send(ctx, errorFrame), ShouldBeNil)
				})
			})
		}),
	)
}

// RQ -> RS: REQUEST_RESPONSE
// RQ -> RS: CANCEL
func TestRequestResponseCanceled(t *testing.T) {
	run(t,
		asClient(func(ctx context.Context, requester *rSocketRequester) {
			Convey("RQ -> When request for response", func() {
				payload, err := requester.RequestResponse(ctx, Text("hello"))
				So(payload, ShouldBeNil)

				So(err, ShouldEqual, context.Canceled)
			})
		}),
		asServer(func(ctx context.Context, cancel context.CancelFunc, requests FrameReceiver, responses FrameSender) {
			Convey("RS -> The request should be sent", func() {
				f, err := requests.Recv(ctx)

				So(err, ShouldBeNil)
				checkFrameHeader(f, 1, frame.TypeRequestResponse, 0)

				requestFrame := f.(*frame.RequestResponseFrame)

				So(requestFrame, ShouldNotBeNil)
				So(string(requestFrame.Data), ShouldEqual, "hello")

				Convey("RS -> RQ: Then cancel the request", func() {
					cancel()
				})
			})
		}),
	)
}

// RQ -> RS: REQUEST_FNF
func TestFireAndForget(t *testing.T) {
	run(t,
		asClient(func(ctx context.Context, requester *rSocketRequester) {
			Convey("RQ -> When send a FireAndForget request", func() {
				So(requester.FireAndForget(ctx, Text("hello").WithMetadata([]byte("world"))), ShouldBeNil)
			})
		}),
		asServer(func(ctx context.Context, cancel context.CancelFunc, requests FrameReceiver, responses FrameSender) {
			Convey("RS -> RQ: Then FireAndForget request should be sent", func() {
				f, err := requests.Recv(ctx)

				So(err, ShouldBeNil)
				checkFrameHeader(f, 1, frame.TypeRequestFireAndForget, frame.FlagMetadata)

				requestFrame := f.(*frame.RequestFireAndForgetFrame)

				So(requestFrame, ShouldNotBeNil)
				So(string(requestFrame.Data), ShouldEqual, "hello")
				So(string(requestFrame.Metadata), ShouldEqual, "world")
			})
		}),
	)
}

// RQ -> RS: METADATA_PUSH
func TestMetadataPush(t *testing.T) {
	run(t,
		asClient(func(ctx context.Context, requester *rSocketRequester) {
			Convey("RQ -> When send metadata push", func() {
				So(requester.MetadataPush(ctx, []byte("hello")), ShouldBeNil)
			})
		}),
		asServer(func(ctx context.Context, cancel context.CancelFunc, requests FrameReceiver, responses FrameSender) {
			Convey("RS -> RQ: Then MetadataPush frame should be sent", func() {
				f, err := requests.Recv(ctx)

				So(err, ShouldBeNil)
				checkFrameHeader(f, 0, frame.TypeMetadataPush, frame.FlagMetadata)

				pushFrame := f.(*frame.MetadataPushFrame)

				So(string(pushFrame.Metadata), ShouldEqual, "hello")
			})
		}),
	)
}
