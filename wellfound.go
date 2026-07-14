package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// fetchWellfoundJobs - HTTP only scraping
func fetchWellfoundJobs() ([]Job, error) {
	url := "https://wellfound.com/role/r/software-engineer"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("unexpected HTTP status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	var jobs []Job

	doc.Find("a[href*='/jobs/']").Each(func(i int, s *goquery.Selection) {
		link, exists := s.Attr("href")
		if !exists || !strings.Contains(link, "/jobs/") {
			return
		}

		title := strings.TrimSpace(s.Text())
		if title == "" || len(title) > 100 {
			return
		}

		if !strings.HasPrefix(link, "http") {
			link = "https://wellfound.com" + link
		}

		slug := link
		if idx := strings.LastIndex(link, "/"); idx > 0 {
			slug = link[idx+1:]
		}

		jobs = append(jobs, Job{
			ID:     "wellfound-" + slug,
			Title:  title,
			Link:   link,
			Source: "Wellfound",
		})
	})

	// Deduplicate
	seen := make(map[string]bool)
	var unique []Job
	for _, j := range jobs {
		if !seen[j.ID] {
			seen[j.ID] = true
			unique = append(unique, j)
		}
	}

	if len(unique) == 0 {
		fmt.Println("  Note: Wellfound requires JavaScript - 0 jobs found via HTTP")
	}

	return unique, nil
}
