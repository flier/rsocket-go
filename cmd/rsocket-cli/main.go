package main

import (
	"errors"
	"net/url"
	"os"

	"github.com/mkideal/cli"
	clix "github.com/mkideal/cli/ext"
	"go.uber.org/zap"

	"github.com/flier/rsocket-go/pkg/rsocket/client"
	"github.com/flier/rsocket-go/pkg/rsocket/frame"
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
	Input           []string          `cli:"i, input" name:"input" usage:"String input, '-' (STDIN) or @path/to/file"`
	Metadata        string            `cli:"m, metadata" name:"metadata" usage:"Metadata input string input or @path/to/file"`
	Setup           string            `cli:"setup" name:"setup" usage:"String input or @path/to/file for setup metadata"`
	Operations      int               `cli:"o, ops" name:"operations" usage:"Operation Count" dft:"1"`
	Timeout         clix.Duration     `cli:"timeout" name:"timeout" usage:"Timeout period"`
	Keepalive       clix.Duration     `cli:"keepalive" name:"keepalive" usage:"Keepalive period"`
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

func main() {
	cli.Run(new(Opts), func(ctx *cli.Context) (err error) {
		var rootLogger *zap.Logger

		opts := ctx.Argv().(*Opts)

		if opts.Debug {
			rootLogger, err = zap.NewDevelopment()
			frame.ReadFrameDumper = os.Stdout
			frame.WriteFrameDumper = os.Stdout
		} else {
			rootLogger, err = zap.NewProduction()
		}

		logger := rootLogger.Named("main")

		if err != nil {
			return err
		}

		var target *url.URL

		targetUri := ctx.Args()[0]
		target, err = url.Parse(targetUri)

		if err != nil {
			logger.Error("invalid target URI", zap.String("uri", targetUri), zap.Error(err))

			return err
		}

		logger.Debug("parsed command line", zap.Reflect("opts", opts), zap.Stringer("target", target))

		if opts.Server {

		} else {
			client, err := client.Dial(target,
				client.WithLogger(rootLogger),
				client.WithKeepalive(opts.Keepalive.Duration),
				client.WithMetadataMimeType(opts.MetadataFmt),
				client.WithDataMimeType(opts.DataFmt))

			if err != nil {
				logger.Error("fail to connect RSocket server", zap.Stringer("target", target), zap.Error(err))

				return err
			}

			logger.Debug("connected to RSocket server", zap.Stringer("target", target), zap.Reflect("client", client))
		}

		return nil
	}, "CLI for RSocket.")
}
