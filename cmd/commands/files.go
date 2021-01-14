package commands

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/gorilla/handlers"
	"github.com/markbates/pkger"
	"github.com/pjvds/tunl/pkg/fallback"
	"github.com/pjvds/tunl/pkg/tunnel"
	"go.uber.org/zap"

	"github.com/urfave/cli/v2"
)

var FilesCommand = &cli.Command{
	Name: "files",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "access-log",
			Value: true,
		},
	},
	Usage:     "Expose a directory via a public http address",
	ArgsUsage: "[dir]",
	Action: func(ctx *cli.Context) error {
		dir := ctx.Args().First()
		if len(dir) == 0 {
			dir = "."
		}

		absDir, err := filepath.Abs(dir)
		if err != nil {
			return cli.Exit("invalid dir: "+err.Error(), 1)
		}

		host := ctx.String("host")
		if len(host) == 0 {
			fmt.Print("Host cannot be empty\nSee --host flag for more information.\n\n")

			cli.ShowCommandHelpAndExit(ctx, ctx.Command.Name, 1)
			return cli.Exit("Host cannot be empty.", 1)
		}

		hostURL, err := url.Parse(host)
		if err != nil {
			fmt.Printf("Host value invalid: %v\nSee --host flag for more information.\n\n", err)

			cli.ShowCommandHelpAndExit(ctx, ctx.Command.Name, 1)
			return nil
		}

		hostnameWithoutPort := hostURL.Hostname()
		if len(hostnameWithoutPort) == 0 {
			fmt.Print("Host hostname cannot be empty, see --host flag for more information.\n\n")

			cli.ShowCommandHelpAndExit(ctx, ctx.Command.Name, 1)
			return nil
		}

		tunnel, err := tunnel.OpenHTTP(ctx.Context, zap.NewNop(), hostURL)
		if err != nil {
			return cli.Exit(err.Error(), 18)
		}

		assets, err := pkger.Open("/assets/favicon")
		if err != nil {
			return cli.Exit(err.Error(), 19)
		}

		handler := fallback.Fallback(http.FileServer(assets), http.FileServer(http.Dir(absDir)))

		if ctx.Bool("access-log") {
			handler = handlers.LoggingHandler(os.Stderr, handler)
		}

		PrintTunnel(tunnel.Address(), absDir)

		go func() {
			for state := range tunnel.StateChanges() {
				println(state)
			}
		}()

		if err := http.Serve(tunnel, handler); err != nil {
			return err
		}

		return nil
	},
}
