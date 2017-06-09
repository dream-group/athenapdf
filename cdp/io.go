package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"github.com/google/uuid"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

var (
	schemeWhitelist []string = []string{"file", "http", "https"}
)

func IsSchemeWhitelisted(scheme string) bool {
	for _, whitelistedScheme := range schemeWhitelist {
		if scheme == whitelistedScheme {
			return true
		}
	}
	return false
}

func HandleInput(uri string) (string, error) {
	// Create URI from stdin payload
	if uri == "-" {
		b, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return "", err
		}
		base64Html := base64.StdEncoding.EncodeToString(b)
		return fmt.Sprintf("data:text/html;base64,%s", base64Html), nil
	}

	u, err := url.Parse(uri)
	if err != nil {
		return "", err
	}

	// Treat URI as a local file path if there is no scheme
	if u.Scheme == "" {
		u.Scheme = "file"
		path, err := filepath.Abs(filepath.Dir(uri))
		if err != nil {
			return "", err
		}
		u.Path = path
	}

	// Security: prevent schemes like `chrome://` from being loaded
	if !IsSchemeWhitelisted(u.Scheme) {
		return "", fmt.Errorf(
			"Input URI contains an invalid scheme. Accepted schemes: %s",
			strings.Join(schemeWhitelist, ", "),
		)
	}

	return u.String(), nil
}

func HandleOutput(b []byte, uri string, dryRun bool) error {
	if dryRun {
		fmt.Printf("\nRunning in dry mode. No output will be created.\n")
		return nil
	}

	switch uri {
	case "-":
		// Write to stdout
		f := bufio.NewWriter(os.Stdout)
		defer f.Flush()
		if _, err := f.Write(b); err != nil {
			return err
		}
	default:
		// Write to file
		// Set a file name if none given
		if uri == "" {
			uri = fmt.Sprintf("%s.pdf", uuid.New())
		}

		if err := ioutil.WriteFile(uri, b, 0644); err != nil {
			return err
		}

		fmt.Printf("\nOutput PDF: %s\n", uri)
	}

	return nil
}
