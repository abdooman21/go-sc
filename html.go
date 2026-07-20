package main

import (
	"net/url"
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

func getURLsFromHTML(htmlBody string, baseURL *url.URL) ([]string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBody))
	if err != nil {
		return nil, err
	}
	urls := []string{}
	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		link, notnil := s.Attr("href")
		if notnil {

			res, err := url.Parse(link)
			if err != nil {
				return
			}
			link := baseURL.ResolveReference(res)

			urls = append(urls, link.String())
		}
	})
	return urls, nil
}

func getImagesFromHTML(htmlBody string, baseURL *url.URL) ([]string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBody))
	if err != nil {
		return nil, err
	}
	images := []string{}
	doc.Find("img[src]").Each(func(_ int, selector *goquery.Selection) {
		src, exsits := selector.Attr("src")
		if exsits {
			img, err := url.Parse(src)
			if err != nil {
				return
			}
			image := baseURL.ResolveReference(img)
			images = append(images, image.String())
		}

	})
	return images, nil
}
