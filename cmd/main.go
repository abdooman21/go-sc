package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// ---------------------------------------------------------------------------
// Stage 1: the search-results JSON API (/AdvancedSearch/Load*ResultPage).
// Gives us every valid ID plus most fields, paginated, with a total count —
// no guessing, no 404s.
// ---------------------------------------------------------------------------

// Record is one row from a Load*ResultPage response.
type Record struct {
	Id             int      `json:"Id"`
	Title          string   `json:"Title"`
	TitleEn        string   `json:"TitleEn"`
	MainResearcher string   `json:"MainResearcher"`
	Views          int      `json:"Views"`
	DetailUrl      string   `json:"DetailUrl"`
	Keywords       string   `json:"Keywords"`
	RatingTotal    int      `json:"RatingTotal"`
	RatingCount    int      `json:"RatingCount"`
	Major          string   `json:"Major"`  // often null -> ""
	Degree         string   `json:"Degree"` // often null -> ""
	Status         string   `json:"Status"` // often null -> ""; real value lives on detail page
	Tags           []string `json:"Tags"`
	AbstractAr     string   `json:"AbstractAr"`
	AbstractEn     string   `json:"AbstractEn"`
	SupervisorName string   `json:"SupervisorName"`
}

// PagedResponse is the envelope every Load*ResultPage endpoint returns.
type PagedResponse struct {
	Result           string   `json:"Result"`
	Records          []Record `json:"Records"`
	TotalRecordCount int      `json:"TotalRecordCount"`
}

// contentType describes one of the five document buckets on the search page.
// Endpoint is the AdvancedSearch controller action name; each Record's
// DetailUrl already carries the right /Research|Thesis|.../Details/{id}
// prefix, so we never hardcode per-type URL paths.
type contentType struct {
	Key      string // used on -types flag
	Label    string // human-readable, used in logs / output
	Endpoint string // AdvancedSearch/{Endpoint}
}

var allContentTypes = []contentType{
	{"research", "Research", "LoadResearchResultPage"},
	{"thesis", "Thesis", "LoadThesisResultPage"},
	{"article", "ScientificArticle", "LoadScientificArticlesResultPage"},
	{"workpaper", "WorkPaper", "LoadWorkPapersResultPage"},
	{"invention", "Invention", "LoadInventionsResultPage"},
}

// fetchPage requests one page of one content type. The POST body mirrors
// the client's empty searchCriteria (i.e. "match everything").
func fetchPage(client *http.Client, baseURL, endpoint string, startIndex, pageSize int) (*PagedResponse, error) {
	q := url.Values{}
	q.Set("startIndex", strconv.Itoa(startIndex))
	q.Set("pageSize", strconv.Itoa(pageSize))
	reqURL := fmt.Sprintf("%s/AdvancedSearch/%s?%s", strings.TrimRight(baseURL, "/"), endpoint, q.Encode())

	form := url.Values{}
	form.Set("Title", "")
	form.Set("MainResearcher", "")
	form.Set("Keywords", "")
	form.Set("Major", "")
	form.Set("ContainerType", "")
	form.Set("ToSearchParameters", "")

	req, err := http.NewRequest(http.MethodPost, reqURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Requested-With", "XMLHttpRequest") // matches jQuery's default AJAX header
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("User-Agent", "go-scraper/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d for %s", resp.StatusCode, reqURL)
	}

	var page PagedResponse
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, fmt.Errorf("decode %s: %w", reqURL, err)
	}
	return &page, nil
}

// fetchAllRecords pages through one content type until every record has
// been collected, or until limit is hit (limit=0 means no limit).
func fetchAllRecords(client *http.Client, baseURL string, ct contentType, pageSize, limit int) ([]Record, error) {
	first, err := fetchPage(client, baseURL, ct.Endpoint, 0, pageSize)
	if err != nil {
		return nil, err
	}

	all := append([]Record{}, first.Records...)
	target := first.TotalRecordCount
	if limit > 0 && limit < target {
		target = limit
	}
	log.Printf("[%s] total=%d target=%d fetched=%d", ct.Label, first.TotalRecordCount, target, len(all))

	if len(all) > target {
		all = all[:target]
	}

	for len(all) < target {
		page, err := fetchPage(client, baseURL, ct.Endpoint, len(all), pageSize)
		if err != nil {
			log.Printf("[%s] page at %d failed: %v", ct.Label, len(all), err)
			break
		}
		if len(page.Records) == 0 {
			break // safety net: stop instead of looping forever
		}
		all = append(all, page.Records...)
		log.Printf("[%s] fetched=%d/%d", ct.Label, len(all), target)
	}

	if len(all) > target {
		all = all[:target]
	}
	return all, nil
}

// ---------------------------------------------------------------------------
// Stage 2: detail pages. Only pulls fields that stage 1's JSON doesn't carry
// (specialty, sub-specialty, real status, arbitration body, publisher, year).
// Selectors confirmed against a live /Research/Details/{id} page.
// ---------------------------------------------------------------------------

type DetailData struct {
	Specialty    string
	SubSpecialty string
	Status       string
	Arbitration  string
	Publisher    string
	Year         string
}

func ExtractDetailData(body io.Reader) (DetailData, error) {
	var data DetailData
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return data, err
	}

	doc.Find(".nds-info-item").Each(func(i int, s *goquery.Selection) {
		label := strings.TrimSpace(s.Find(".nds-info-label").Text())
		value := cleanText(s.Find(".nds-info-value").Text())

		switch label {
		case "التخصص":
			data.Specialty = value
		case "التخصص الدقيق":
			data.SubSpecialty = value
		case "الحالة":
			data.Status = value
		case "جهة التحكيم":
			data.Arbitration = value
		case "دار النشر":
			data.Publisher = value
		case "سنة النشر":
			data.Year = value
		}
	})

	return data, nil
}

func cleanText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", " ")
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", " ")
	return strings.Join(strings.Fields(text), " ")
}

func fetchDetail(client *http.Client, baseURL, detailURL string) (DetailData, error) {
	full := detailURL
	if strings.HasPrefix(detailURL, "/") {
		full = strings.TrimRight(baseURL, "/") + detailURL
	}

	req, err := http.NewRequest(http.MethodGet, full, nil)
	if err != nil {
		return DetailData{}, err
	}
	req.Header.Set("User-Agent", "go-scraper/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return DetailData{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return DetailData{}, fmt.Errorf("status %d for %s", resp.StatusCode, full)
	}

	return ExtractDetailData(resp.Body)
}

// ---------------------------------------------------------------------------
// Merged output
// ---------------------------------------------------------------------------

type FullRecord struct {
	Id           int      `json:"id"`
	Type         string   `json:"type"`
	TitleAr      string   `json:"title_ar"`
	TitleEn      string   `json:"title_en"`
	Researcher   string   `json:"researcher"`
	Supervisor   string   `json:"supervisor,omitempty"`
	Specialty    string   `json:"specialty,omitempty"`
	SubSpecialty string   `json:"sub_specialty,omitempty"`
	AbstractAr   string   `json:"abstract_ar,omitempty"`
	AbstractEn   string   `json:"abstract_en,omitempty"`
	Status       string   `json:"status,omitempty"`
	Arbitration  string   `json:"arbitration,omitempty"`
	Publisher    string   `json:"publisher,omitempty"`
	Year         string   `json:"year,omitempty"`
	Keywords     string   `json:"keywords,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Views        int      `json:"views"`
	DetailUrl    string   `json:"detail_url"`
}

func mergeRecord(typeLabel string, r Record, d DetailData) FullRecord {
	status := r.Status
	if status == "" {
		status = d.Status
	}
	return FullRecord{
		Id:           r.Id,
		Type:         typeLabel,
		TitleAr:      r.Title,
		TitleEn:      r.TitleEn,
		Researcher:   r.MainResearcher,
		Supervisor:   r.SupervisorName,
		Specialty:    d.Specialty,
		SubSpecialty: d.SubSpecialty,
		AbstractAr:   cleanText(r.AbstractAr),
		AbstractEn:   cleanText(r.AbstractEn),
		Status:       status,
		Arbitration:  d.Arbitration,
		Publisher:    d.Publisher,
		Year:         d.Year,
		Keywords:     r.Keywords,
		Tags:         r.Tags,
		Views:        r.Views,
		DetailUrl:    r.DetailUrl,
	}
}

func writeOutput(path string, results []FullRecord) {
	f, err := os.Create(path)
	if err != nil {
		log.Fatalf("create output file: %v", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(results); err != nil {
		log.Fatalf("write output: %v", err)
	}
	log.Printf("wrote %d records to %s", len(results), path)
}

type stagedItem struct {
	ct  contentType
	rec Record
}

func main() {
	baseURL := flag.String("base", "https://mysite", "Base site URL, no trailing slash")
	pageSize := flag.Int("page-size", 250, "Records per stage-1 page request (10/25/50/100/250 are known-valid)")
	typesFlag := flag.String("types", "research,thesis,article,workpaper,invention", "Comma-separated content types to fetch")
	limit := flag.Int("limit", 0, "Max records per type (0 = no limit; use a small number to test first)")
	skipDetails := flag.Bool("skip-details", false, "Skip stage 2 (detail pages); output stage-1 fields only")
	detailConcurrency := flag.Int("detail-concurrency", 15, "Concurrent detail-page requests per batch")
	detailDelay := flag.Int("detail-delay", 1, "Seconds to sleep between detail-page batches")
	output := flag.String("output", "records.json", "Output JSON file path")
	flag.Parse()

	wanted := map[string]bool{}
	for _, k := range strings.Split(*typesFlag, ",") {
		wanted[strings.TrimSpace(k)] = true
	}

	client := &http.Client{Timeout: 20 * time.Second}

	// ---- Stage 1: page through the JSON API for every requested type ----
	var staged []stagedItem
	for _, ct := range allContentTypes {
		if !wanted[ct.Key] {
			continue
		}
		records, err := fetchAllRecords(client, *baseURL, ct, *pageSize, *limit)
		if err != nil {
			log.Printf("[%s] stage 1 failed: %v", ct.Label, err)
			continue
		}
		for _, r := range records {
			staged = append(staged, stagedItem{ct, r})
		}
	}
	log.Printf("stage 1 done: %d records total", len(staged))

	if *skipDetails {
		results := make([]FullRecord, 0, len(staged))
		for _, item := range staged {
			results = append(results, mergeRecord(item.ct.Label, item.rec, DetailData{}))
		}
		writeOutput(*output, results)
		return
	}

	// ---- Stage 2: fetch each record's detail page in batches ----
	results := make([]FullRecord, len(staged))
	var wg sync.WaitGroup

	for i := 0; i < len(staged); i += *detailConcurrency {
		end := i + *detailConcurrency
		if end > len(staged) {
			end = len(staged)
		}

		log.Printf("--- detail batch %d-%d of %d ---", i, end, len(staged))

		for offset := i; offset < end; offset++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				item := staged[idx]

				detail, err := fetchDetail(client, *baseURL, item.rec.DetailUrl)
				if err != nil {
					log.Printf("[%s id=%d] detail fetch failed: %v", item.ct.Label, item.rec.Id, err)
				}

				// Safe without a mutex: every goroutine writes a distinct index.
				results[idx] = mergeRecord(item.ct.Label, item.rec, detail)
			}(offset)
		}

		wg.Wait()

		if end < len(staged) {
			time.Sleep(time.Duration(*detailDelay) * time.Second)
		}
	}

	writeOutput(*output, results)
}
