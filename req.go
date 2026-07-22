package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func getHTML(rawURL string) (string, error) {

	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "BootCrawler/1.0")
	req.Header.Set("Accept", "text/html")

	client := &http.Client{Timeout: time.Second * 10}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("error status %d ", resp.StatusCode)
	}
	if !strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
		return "", fmt.Errorf("resp not html cont type %s", resp.Header.Get("Content-Type"))
	}
	html, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(html), nil
}

func crawlPage(rawBaseURL, rawCurrentURL string, pages map[string]int) {
	curURL, err := url.Parse(rawCurrentURL)
	if err != nil {
		return
	}

	baseURL, err := url.Parse(rawBaseURL)
	if err != nil {
		return
	}

	if curURL.Hostname() != baseURL.Hostname() {
		return
	}

	normalized, err := NormalizeURL(rawCurrentURL)
	if err != nil {
		return
	}

	if _, ok := pages[normalized]; ok {
		pages[normalized]++
		return
	}

	pages[normalized] = 1

	html, err := getHTML(rawCurrentURL)
	if err != nil {
		return
	}

	links, err := getURLsFromHTML(html, curURL)
	if err != nil {
		return
	}

	for _, link := range links {
		crawlPage(rawBaseURL, link, pages)
	}
}
