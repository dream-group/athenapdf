package main

import (
	"fmt"
	"github.com/urfave/cli"
	"os"
	"strings"
)

const (
	appName        = "athenapdf"
	appVersion     = "3.0.0-b"
	appDescription = "convert (M)HTML to PDF using headless Chromium / Blink (DevTools Protocol)"

	defaultPageHeight = 11
	defaultPageWidth  = 8.5
	defaultPageMargin = 0.4
)

func main() {
	app := cli.NewApp()
	app.Name = appName
	app.Version = appVersion
	app.Usage = appDescription
	app.UsageText = fmt.Sprintf("%s [options...] <input> <output>", app.Name)

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug, D",
			Usage: "show verbose logging",
		},
		cli.BoolFlag{
			Name:  "dry-run",
			Usage: "do not render the PDF output, useful for testing",
		},
		cli.StringFlag{
			Name:  "proxy",
			Usage: "use a proxy `SERVER` for HTTP(S) requests (only works in non-client-server mode, default)",
		},
		cli.StringFlag{
			Name:  "server",
			Usage: "run in client-server mode by connecting to a local running instance of Chromium's Remote Debugging Protocol",
		},
		cli.Uint64Flag{
			Name:  "timeout",
			Usage: "seconds to wait for the page to load before timing out",
			Value: 60,
		},
		cli.BoolFlag{
			Name:  "no-background",
			Usage: "do not print background graphics",
		},
		cli.BoolFlag{
			Name:  "no-cache",
			Usage: "do not use cache for any request",
		},
		cli.BoolFlag{
			Name:  "no-javascript",
			Usage: "do not execute JavaScript",
		},
		cli.StringFlag{
			Name:  "orientation, O",
			Usage: "orientation of PDF, Landscape or Portrait",
			Value: "Portrait",
		},
		cli.Float64Flag{
			Name:  "margin-bottom, B",
			Usage: "bottom margin of PDF in inches",
			Value: defaultPageMargin,
		},
		cli.Float64Flag{
			Name:  "margin-left, L",
			Usage: "left margin of PDF in inches",
			Value: defaultPageMargin,
		},
		cli.Float64Flag{
			Name:  "margin-right, R",
			Usage: "right margin of PDF in inches",
			Value: defaultPageMargin,
		},
		cli.Float64Flag{
			Name:  "margin-top, T",
			Usage: "top margin of PDF in inches",
			Value: defaultPageMargin,
		},
		cli.Float64Flag{
			Name:  "page-height, H",
			Usage: "height of PDF in inches",
			Value: defaultPageHeight,
		},
		cli.Float64Flag{
			Name:  "page-width, W",
			Usage: "width of PDF in inches",
			Value: defaultPageWidth,
		},
		cli.Float64Flag{
			Name:  "scale, S",
			Usage: "scale of PDF rendering",
			Value: 1,
		},
		cli.StringSliceFlag{
			Name:  "cookie",
			Usage: "set an additional `key:value` cookie, the value must be URL encoded (repeatable)",
		},
		cli.StringSliceFlag{
			Name:  "custom-header",
			Usage: "set an additional `key:value` HTTP header (repeatable)",
		},
		cli.StringSliceFlag{
			Name: "js-plugin",
			Usage: fmt.Sprintf(
				"JavaScript plugin to execute on page load (repeatable, pre-installed options: %s)",
				strings.Join(availableJSPlugins, ", "),
			),
		},
	}

	app.Action = AppWithErrorHandler

	app.Run(os.Args)
}
