package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/urfave/cli"
	"github.com/wirepair/gcd"
	"github.com/wirepair/gcd/gcdapi"
	"io/ioutil"
	"log"
	"net/url"
	"time"
)

var (
	defaultFlags = []string{
		"--allow-running-insecure-content",
		"--disable-gpu",
		"--disable-new-tab-first-run",
		"--disable-notifications",
		"--headless",
		"--ignore-certificate-errors",
		"--no-default-browser-check",
		"--no-first-run",
	}
)

type ExitFunc func() error

func AppWithErrorHandler(app *cli.Context) error {
	startTime := time.Now()
	if app.Bool("debug") {
		defer func() {
			fmt.Printf("\nProcessing time: %s\n", time.Since(startTime).String())
		}()
	}
	return RawAppHandler(app)
}

func RawAppHandler(app *cli.Context) error {
	input := app.Args().Get(0)
	output := app.Args().Get(1)

	if input == "" {
		fmt.Printf("\nNo input URI given. Set it to `-` if you are piping via stdin.\n\n")
		cli.ShowAppHelp(app)
		return nil
	}

	input, err := HandleInput(input)
	if err != nil {
		return err
	}

	t, exit, err := NewAutoTarget(app.String("server"), app.String("proxy"))
	if err != nil {
		return err
	}

	defer func() {
		if err := exit(); err != nil {
			log.Fatalln(err)
		}
	}()

	t.Debug(app.Bool("debug"))
	t.DebugEvents(app.Bool("debug"))

	if err := ConfigureTargetFromCLI(t, app, input); err != nil {
		return err
	}

	fid, err := t.Page.Navigate(input, "", "")
	if err != nil {
		return err
	}

	pageReady := make(chan bool, 1)
	t.Subscribe("Page.loadEventFired", func(_ *gcd.ChromeTarget, _ []byte) {
		pageReady <- true
	})

	// Detect errors in page load
	pageFailed := make(chan string, 1)
	t.Subscribe("Network.loadingFailed", func(_ *gcd.ChromeTarget, b []byte) {
		var v gcdapi.NetworkLoadingFailedEvent
		if err := json.Unmarshal(b, &v); err != nil {
			if app.Bool("debug") {
				log.Println(err)
			}
			return
		}
		if v.Params.RequestId == fid {
			pageFailed <- v.Params.ErrorText
		}
	})

	loadedPlugins := make(chan string, 1)
	go func() {
		p, err := GetJsPlugins(app.StringSlice("js-plugin")...)
		if err != nil {
			log.Fatalln(err)
		}
		loadedPlugins <- p
	}()

	select {
	case <-time.After(time.Second * time.Duration(app.Uint64("timeout"))):
		return fmt.Errorf("Timeout waiting for the page to load.")
	case errorText := <-pageFailed:
		return fmt.Errorf("Failed to load the page: %s", errorText)
	case <-pageReady:
	}

	evaluateParams := gcdapi.RuntimeEvaluateParams{Expression: <-loadedPlugins}
	if _, _, err := t.Runtime.EvaluateWithParams(&evaluateParams); err != nil {
		return err
	}

	pdfParams := gcdapi.PagePrintToPDFParams{
		Landscape:       app.String("orientation") == "Landscape",
		PrintBackground: !app.Bool("no-background"),
		Scale:           app.Float64("scale"),
		PaperWidth:      app.Float64("page-width"),
		PaperHeight:     app.Float64("page-height"),
		MarginTop:       app.Float64("margin-top"),
		MarginBottom:    app.Float64("margin-bottom"),
		MarginLeft:      app.Float64("margin-left"),
		MarginRight:     app.Float64("margin-right"),
	}
	base64String, err := t.Page.PrintToPDFWithParams(&pdfParams)
	if err != nil {
		return err
	}

	b, err := base64.StdEncoding.DecodeString(base64String)
	if err := HandleOutput(b, output, app.Bool("dry-run")); err != nil {
		return err
	}

	return nil
}

func ConfigureTargetFromCLI(t *gcd.ChromeTarget, app *cli.Context, input string) error {
	userAgent := fmt.Sprintf("%s/%s", app.App.Name, app.App.Version)
	if _, err := t.Network.SetUserAgentOverride(userAgent); err != nil {
		return err
	}

	if _, err := t.Network.SetCacheDisabled(app.Bool("no-cache")); err != nil {
		return err
	}

	if _, err := t.Emulation.SetScriptExecutionDisabled(app.Bool("no-javascript")); err != nil {
		return err
	}

	if input != "" {
		for name, value := range getKeyValueMap(app.StringSlice("cookie")...) {
			cookies := gcdapi.NetworkSetCookieParams{
				Url:   input,
				Name:  name,
				Value: fmt.Sprintf("%s", value),
			}
			if _, err := t.Network.SetCookieWithParams(&cookies); err != nil {
				return err
			}
		}
	}

	httpHeaders := getKeyValueMap(app.StringSlice("custom-header")...)
	if _, err := t.Network.SetExtraHTTPHeaders(httpHeaders); err != nil {
		return err
	}

	return nil
}

func NewAutoTarget(server string, proxy string) (*gcd.ChromeTarget, ExitFunc, error) {
	var exitFunc ExitFunc = func() error { return nil }

	c, clientExit, err := StartCDP(server, proxy)
	if err != nil {
		return nil, exitFunc, err
	}

	t, targetExit, err := StartTarget(c)
	if err != nil {
		return nil, exitFunc, err
	}

	if server == "" {
		exitFunc = clientExit
	} else {
		exitFunc = targetExit
	}

	return t, exitFunc, nil
}

func StartCDP(server string, proxy string) (*gcd.Gcd, ExitFunc, error) {
	var exitFunc ExitFunc = func() error { return nil }

	// Get a random port to avoid conflicting instances
	randomPort, err := getRandomPort()
	if err != nil {
		return nil, exitFunc, err
	}

	// Create a random directory for user data
	randomDir, err := ioutil.TempDir("", appName)
	if err != nil {
		return nil, exitFunc, err
	}

	client := gcd.NewChromeDebugger()

	if server == "" {
		// Create a new CDP process if no instance specified
		client.AddFlags(defaultFlags)
		if proxy != "" {
			client.AddFlags([]string{"--proxy-server=" + proxy})
		}
		client.StartProcess(getChromePath(), randomDir, randomPort)
		exitFunc = func() error { return client.ExitProcess() }
	} else {
		// Connect to an existing instance if specified
		u, err := url.Parse(server)
		if err != nil {
			return nil, exitFunc, err
		}
		client.ConnectToInstance(u.Hostname(), u.Port())
	}

	return client, exitFunc, nil
}

func StartTarget(client *gcd.Gcd) (*gcd.ChromeTarget, ExitFunc, error) {
	var exitFunc ExitFunc = func() error { return nil }

	t, err := client.NewTab()
	if err != nil {
		return nil, exitFunc, err
	}

	t.CSS.Enable()
	t.DOM.Enable()
	t.Network.Enable(-1, -1)
	t.Page.Enable()
	t.Runtime.Enable()
	t.Log.Enable()

	exitFunc = func() error { return client.CloseTab(t) }

	return t, exitFunc, nil
}
