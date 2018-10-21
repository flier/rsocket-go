package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"strings"

	"github.com/2tvenom/cbor"
	"github.com/flier/rsocket-go/pkg/rsocket/client"
	"github.com/flier/rsocket-go/pkg/rsocket/frame"
	"github.com/flier/rsocket-go/pkg/rsocket/proto"
	"github.com/mkideal/cli"
	clix "github.com/mkideal/cli/ext"
	"go.uber.org/zap"
)

type Opts struct {
	cli.Helper

	Headers         map[string]string `cli:"H, header" name:"name=value" usage:"Request Response"`
	RequestResponse bool              `cli:"request" usage:"Request Response"`
	FireAndForget   bool              `cli:"fnf" usage:"Fire and Forget"`
	Stream          bool              `cli:"stream" usage:"Request Stream"`
	Channel         bool              `cli:"channel" usage:"Request Channel"`
	MetadataPush    bool              `cli:"metadataPush" usage:"Metadata Push"`
	Server          bool              `cli:"s, server" usage:"Start server instead of client"`
	Debug           bool              `cli:"d, debug" usage:"Debug Output"`
	MetadataFmt     string            `cli:"metadataFmt" name:"mimeType" usage:"Metadata Format" dft:"json"`
	DataFmt         string            `cli:"dataFmt" name:"mimeType" usage:"Data Format" dft:"binary"`
	Input           []string          `cli:"i, input" name:"input" usage:"String input, '-' (STDIN) or @path/to/file" dft:"-"`
	Metadata        string            `cli:"m, metadata" name:"metadata" usage:"Metadata input string input or @path/to/file"`
	Setup           string            `cli:"setup" name:"setup" usage:"String input or @path/to/file for setup metadata"`
	Operations      int               `cli:"o, ops" name:"operations" usage:"Operation Count" dft:"1"`
	Timeout         clix.Duration     `cli:"timeout" name:"timeout" usage:"Timeout period" dft:"500ms"`
	Keepalive       clix.Duration     `cli:"keepalive" name:"keepalive" usage:"Keepalive period" dft:"2s"`
	RequestN        int               `cli:"r, requestn" name:"requests" usage:"Request N credits"`
}

// Validate check the command line options
func (opts *Opts) Validate(ctx *cli.Context) error {
	if ctx.NArg() == 0 {
		return errors.New("missing target URL")
	}

	if ctx.NArg() > 1 {
		return errors.New("too many target URL")
	}

	return nil
}

func (opts *Opts) parseSetupData() (*proto.Payload, error) {
	switch {
	case strings.HasPrefix(opts.Setup, "@"):
		data, err := ioutil.ReadFile(opts.Setup[1:])

		if err != nil {
			return nil, err
		}

		return proto.Bytes(data), nil

	case opts.Setup == "":
		return new(proto.Payload), nil

	default:
		return proto.Text(opts.Setup), nil
	}
}

func (opts *Opts) buildMetadata() (proto.Metadata, error) {
	if len(opts.Metadata) > 0 {
		return proto.Metadata(opts.Metadata), nil
	}

	if len(opts.Headers) > 0 {
		switch opts.MetadataFmt {
		case "json":
			data, err := json.Marshal(opts.Headers)

			if err != nil {
				return nil, err
			}

			return proto.Metadata(data), nil

		case "cbor":
			var buf bytes.Buffer

			encoder := cbor.NewEncoder(&buf)

			_, err := encoder.Marshal(opts.Headers)

			if err != nil {
				return nil, err
			}

			return proto.Metadata(buf.Bytes()), nil

		default:
			return nil, errors.New("headers not supported with mimetype: " + opts.MetadataFmt)
		}
	}

	return nil, nil
}

func (opts *Opts) PayloadStream(ctx context.Context) *proto.PayloadStream {
	c := make(chan *proto.Result)

	go func() error {
		defer close(c)

		readLines := func(r io.Reader) error {
			br := bufio.NewReader(r)

			for {
				line, err := br.ReadString('\n')

				if err != nil {
					return err
				}

				select {
				case <-ctx.Done():
					return ctx.Err()
				case c <- proto.Ok(proto.Text(line)):
				}
			}
		}

		for _, input := range opts.Input {
			input = strings.TrimSpace(input)

			switch {
			case input == "-":
				if err := readLines(os.Stdin); err != nil {
					return err
				}

			case strings.HasPrefix(input, "@"):
				f, err := os.Open(input[1:])

				if err != nil {
					return err
				}

				defer f.Close()

				if err := readLines(f); err != nil {
					return err
				}

			default:
				select {
				case <-ctx.Done():
					return ctx.Err()
				case c <- proto.Ok(proto.Text(input)):
				}
			}
		}

		return nil
	}()

	return &proto.PayloadStream{C: c}
}

func main() {
	cli.Run(new(Opts), func(cmdline *cli.Context) (err error) {
		var rootLogger *zap.Logger

		opts := cmdline.Argv().(*Opts)

		if opts.Debug {
			rootLogger, err = zap.NewDevelopment()
			frame.ReadFrameDumper = os.Stdout
			frame.WriteFrameDumper = os.Stdout
		} else {
			rootLogger, err = zap.NewProduction()
		}

		logger := rootLogger.Named("main")

		if err != nil {
			return
		}

		var target *url.URL

		targetUri := cmdline.Args()[0]
		target, err = url.Parse(targetUri)

		if err != nil {
			logger.Error("invalid target URI", zap.String("uri", targetUri), zap.Error(err))

			return
		}

		logger.Debug("parsed command line", zap.Reflect("opts", opts), zap.Stringer("target", target))

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		if opts.Server {

		} else {
			setupPayload, err := opts.parseSetupData()

			if err != nil {
				return err
			}

			client, err := client.DialContext(ctx, target,
				client.WithLogger(rootLogger),
				client.WithKeepalive(opts.Keepalive.Duration),
				client.WithMaxLifetime(opts.Keepalive.Duration*2),
				client.WithMetadataMimeType(opts.MetadataFmt),
				client.WithDataMimeType(opts.DataFmt),
				client.WithSetupPayload(setupPayload),
			)

			if err != nil {
				logger.Error("fail to connect RSocket server", zap.Stringer("target", target), zap.Error(err))

				return err
			}

			logger.Debug("connected to RSocket server", zap.Stringer("target", target), zap.Reflect("client", client))

			requester := client.Requester()

			switch {
			case opts.FireAndForget:
				payload, err := opts.PayloadStream(ctx).Recv(ctx)

				if err != nil {
					return err
				}

				return requester.FireAndForget(ctx, payload)

			case opts.MetadataPush:
				return requester.MetadataPush(ctx, proto.Metadata(opts.Metadata))

			case opts.RequestResponse:
				payload, err := opts.PayloadStream(ctx).Recv(ctx)

				if err != nil {
					return err
				}

				payload, err = requester.RequestResponse(ctx, payload)

				if err != nil {
					return err
				}

				println(payload.Text())

			case opts.Stream:
				payload, err := opts.PayloadStream(ctx).Recv(ctx)

				if err != nil {
					return err
				}

				responses, err := requester.RequestStream(ctx, payload)

				if err != nil {
					return err
				}

				for {
					payload, err := responses.Recv(ctx)

					if err != nil {
						return err
					}

					if payload == nil {
						break
					}

					println(payload.Text())
				}

			case opts.Channel:
				responses, err := requester.RequestChannel(ctx, opts.PayloadStream(ctx))

				if err != nil {
					return err
				}

				for {
					payload, err := responses.Recv(ctx)

					if err != nil {
						return err
					}

					if payload == nil {
						break
					}

					println(payload.Text())
				}
			}
		}

		return nil
	}, "CLI for RSocket.")
}
