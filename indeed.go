package main

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// fetchIndeedJobs fetches jobs from multiple Indeed RSS feeds
// Falls back to HTML scraping if RSS fails
func fetchIndeedJobs(rssURLs []string) ([]Job, error) {
	if len(rssURLs) == 0 {
		rssURLs = []string{"https://www.indeed.com/jobs?q=backend+go&l=India"}
	}

	var allJobs []Job
	var failures []string

	for _, url := range rssURLs {
		// Convert RSS URLs to regular search URLs
		searchURL := strings.Replace(url, "/rss?", "/jobs?", 1)

		jobs, err := scrapeIndeedPage(searchURL)
		if err != nil {
			fmt.Printf("  Warning: Could not scrape %s: %v\n", searchURL, err)
			failures = append(failures, err.Error())
			continue
		}
		allJobs = append(allJobs, jobs...)
	}
	if len(allJobs) == 0 && len(failures) > 0 {
		return nil, fmt.Errorf("all Indeed searches failed: %s", strings.Join(failures, "; "))
	}

	return allJobs, nil
}

func scrapeIndeedPage(url string) ([]Job, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	// Use a modern User-Agent to reduce Cloudflare blocking
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Referer", "https://www.google.com/")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "cross-site")

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
	jobIDRegex := regexp.MustCompile(`jk=([a-f0-9]+)`)

	// Indeed job cards
	doc.Find(".job_seen_beacon, .jobsearch-ResultsList > li, .resultContent").Each(func(i int, s *goquery.Selection) {
		// Find title
		title := s.Find("h2.jobTitle span, .jobTitle, a[data-jk]").First().Text()
		title = strings.TrimSpace(title)

		if title == "" {
			return
		}

		// Find link
		link, _ := s.Find("a[href*='/rc/clk'], a[href*='viewjob'], a[data-jk]").First().Attr("href")
		if link == "" {
			link, _ = s.Find("a").First().Attr("href")
		}

		if link == "" {
			return
		}

		// Extract job ID
		var jobID string
		if matches := jobIDRegex.FindStringSubmatch(link); len(matches) > 1 {
			jobID = matches[1]
		} else {
			jobID = fmt.Sprintf("%d", i)
		}

		// Make absolute URL
		if !strings.HasPrefix(link, "http") {
			link = "https://www.indeed.com" + link
		}

		jobs = append(jobs, Job{
			ID:     "indeed-" + jobID,
			Title:  title,
			Link:   link,
			Source: "Indeed",
		})
	})

	return jobs, nil
}
