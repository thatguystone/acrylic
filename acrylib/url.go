package acrylib

import "strings"

func isRemoteURL(u string) bool {
	return strings.HasPrefix(u, "http://") ||
		strings.HasPrefix(u, "https://") ||
		strings.HasPrefix(u, "//")
}

func checkURLProtocol(u string) string {
	if strings.HasPrefix(u, "//") {
		return "http:" + u
	}

	return u
}
