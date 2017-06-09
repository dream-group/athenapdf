package main

import (
	"bytes"
	"fmt"
	"strings"
)

var (
	availableJSPlugins []string = []string{
		"domdistiller",
		"no-css-media",
	}
	defaultJSPlugins []string = []string{
		"no-css-media",
	}
)

func GetJsPlugins(pluginsToLoad ...string) (string, error) {
	pluginsToLoad = append(pluginsToLoad, defaultJSPlugins...)

	var buf bytes.Buffer

	for _, pluginName := range pluginsToLoad {
		plugin, err := GetJsPlugin(pluginName)
		if err != nil {
			return "", err
		}
		if _, err := buf.Write(plugin); err != nil {
			return "", err
		}
	}

	return buf.String(), nil
}

func GetJsPlugin(pluginName string) ([]byte, error) {
	if !PluginExists(pluginName) {
		return nil, fmt.Errorf(
			"Plugin `%s` does not exist. Available plugins: %s",
			pluginName,
			strings.Join(availableJSPlugins, ", "),
		)
	}

	b, err := Asset(fmt.Sprintf("js-plugins/%s.js", pluginName))
	if err != nil {
		return nil, err
	}

	return b, nil
}

func PluginExists(pluginName string) bool {
	for _, availablePlugin := range availableJSPlugins {
		if pluginName == availablePlugin {
			return true
		}
	}

	return false
}
