package main

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func getHeadingFromHTML(html string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return ""
	}

	h1 := doc.Find("h1").First()
	if h1.Length() > 0 {
		return h1.Text()
	}

	h2 := doc.Find("h2").First()
	if h2.Length() > 0 {
		return h2.Text()
	}

	return ""
}

func getFirstParagraphFromHTML(html string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return ""
	}

	mainP := doc.Find("main p").First()
	if mainP.Length() > 0 {
		return mainP.Text()
	}

	p := doc.Find("p").First()
	if p.Length() > 0 {
		return p.Text()
	}

	return ""
}
