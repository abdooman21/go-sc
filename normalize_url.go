package main

import (
	"net/url"
	"strings"
)

// inputURL: "https://www.boot.dev/blog/path",
// expected: "www.boot.dev/blog/path",
func NormalizeURL(rawURL string) (string, error) {
	parts, err := url.Parse(rawURL)

	path := strings.TrimSuffix(parts.Path, "/")
	fullurl := parts.Host + path

	return fullurl, err
}
