package main

import (
	"net"
	"strings"
)

func getRandomPort() (string, error) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return "", err
	}
	defer l.Close()

	_, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		return "", err
	}

	return port, nil
}

func getKeyValueMap(kvStrings ...string) map[string]interface{} {
	keyValueMap := make(map[string]interface{}, len(kvStrings))
	for _, kvString := range kvStrings {
		kvParts := strings.SplitN(kvString, ":", 2)
		// Skip invalid key:value
		if len(kvParts) != 2 {
			continue
		}
		keyValueMap[kvParts[0]] = kvParts[1]
	}
	return keyValueMap
}
